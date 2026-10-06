package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// broker reparte avisos en tiempo real a las pestañas abiertas (Server-Sent Events).
// Vive en memoria: con varias instancias de la API, cada una solo avisa a sus conexiones.
type broker struct {
	mu   sync.Mutex
	subs map[*subscriber]struct{}
}

type subscriber struct {
	userID int64
	staff  bool
	ch     chan []byte
}

func newBroker() *broker {
	return &broker{subs: map[*subscriber]struct{}{}}
}

func (b *broker) subscribe(userID int64, staff bool) *subscriber {
	s := &subscriber{userID: userID, staff: staff, ch: make(chan []byte, 32)}
	b.mu.Lock()
	b.subs[s] = struct{}{}
	b.mu.Unlock()
	return s
}

func (b *broker) unsubscribe(s *subscriber) {
	b.mu.Lock()
	delete(b.subs, s)
	b.mu.Unlock()
}

type streamEvent struct {
	Type     string `json:"type"` // ticket, notification
	TicketID int64  `json:"ticket_id,omitempty"`
}

// publish envía los avisos de una operación ya confirmada. Si un cliente va atrasado,
// se descarta el aviso: al reconectar, el frontend vuelve a cargar los datos.
func (b *broker) publish(f fanout) {
	if len(f.users) == 0 && len(f.tickets) == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	send := func(s *subscriber, ev streamEvent) {
		data, _ := json.Marshal(ev)
		select {
		case s.ch <- data:
		default:
		}
	}
	for s := range b.subs {
		for _, t := range f.tickets {
			if s.staff || (!t.staffOnly && s.userID == t.requester) {
				send(s, streamEvent{Type: "ticket", TicketID: t.id})
			}
		}
		for _, id := range f.users {
			if s.userID == id {
				send(s, streamEvent{Type: "notification"})
			}
		}
	}
}

// maxStreamDuration obliga a reconectar de vez en cuando, y así a revalidar el token.
const maxStreamDuration = 30 * time.Minute

// stream mantiene abierta la conexión y envía un evento por cada cambio relevante.
func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)
	// Esta conexión dura mucho más que los plazos normales del servidor.
	_ = rc.SetReadDeadline(time.Time{})
	_ = rc.SetWriteDeadline(time.Time{})

	claims := claimsFrom(r)
	sub := s.broker.subscribe(claims.UserID(), claims.IsStaff())
	defer s.broker.unsubscribe(sub)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "retry: 5000\n\n")
	if err := rc.Flush(); err != nil {
		slog.Warn("el servidor no admite streaming", "error", err)
		return
	}

	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	deadline := time.NewTimer(maxStreamDuration)
	defer deadline.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-deadline.C:
			return
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
		case data := <-sub.ch:
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		if err := rc.Flush(); err != nil {
			return
		}
	}
}
