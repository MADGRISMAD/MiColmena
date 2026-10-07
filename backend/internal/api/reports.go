package api

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// reportRange lee ?from=AAAA-MM-DD&to=AAAA-MM-DD&tz=America/Mexico_City.
// Por defecto, los últimos 30 días en UTC. "to" es inclusivo.
type reportRange struct {
	from, to time.Time // to es exclusivo (el día siguiente a las 00:00)
	tz       string
}

func parseRange(w http.ResponseWriter, r *http.Request) (reportRange, bool) {
	q := r.URL.Query()
	tz := q.Get("tz")
	if tz == "" {
		tz = "UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		writeError(w, http.StatusBadRequest, "zona horaria inválida")
		return reportRange{}, false
	}
	today := time.Now().In(loc)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)
	rr := reportRange{from: today.AddDate(0, 0, -29), to: today.AddDate(0, 0, 1), tz: tz}

	parse := func(key string) (time.Time, bool) {
		v := q.Get(key)
		if v == "" {
			return time.Time{}, true
		}
		d, err := time.ParseInLocation("2006-01-02", v, loc)
		if err != nil {
			writeError(w, http.StatusBadRequest, key+" debe tener el formato AAAA-MM-DD")
			return time.Time{}, false
		}
		return d, true
	}
	from, ok := parse("from")
	if !ok {
		return rr, false
	}
	to, ok := parse("to")
	if !ok {
		return rr, false
	}
	if !from.IsZero() {
		rr.from = from
	}
	if !to.IsZero() {
		rr.to = to.AddDate(0, 0, 1)
	}
	if !rr.from.Before(rr.to) {
		writeError(w, http.StatusBadRequest, "from debe ser anterior o igual a to")
		return rr, false
	}
	if rr.to.Sub(rr.from) > 366*24*time.Hour {
		writeError(w, http.StatusBadRequest, "el rango no puede superar un año")
		return rr, false
	}
	return rr, true
}

type dailyCount struct {
	Date     string `json:"date"`
	Created  int    `json:"created"`
	Resolved int    `json:"resolved"`
}

type agentReport struct {
	ID                 int64    `json:"id"`
	Name               string   `json:"name"`
	OpenAssigned       int      `json:"open_assigned"`
	Resolved           int      `json:"resolved"`
	AvgResolutionHours *float64 `json:"avg_resolution_hours"`
	Good               int      `json:"satisfaction_good"`
	Bad                int      `json:"satisfaction_bad"`
}

type nameCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type reportResponse struct {
	From                  string        `json:"from"`
	To                    string        `json:"to"`
	Created               int           `json:"created"`
	Resolved              int           `json:"resolved"`
	AvgFirstResponseHours *float64      `json:"avg_first_response_hours"`
	AvgResolutionHours    *float64      `json:"avg_resolution_hours"`
	FirstResponseSLAMet   *float64      `json:"first_response_sla_met"`
	Good                  int           `json:"satisfaction_good"`
	Bad                   int           `json:"satisfaction_bad"`
	Daily                 []dailyCount  `json:"daily"`
	Agents                []agentReport `json:"agents"`
	Categories            []nameCount   `json:"categories"`
}

// report resume la actividad del rango: volumen por día, tiempos, SLA, satisfacción y desempeño por agente.
func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	rr, ok := parseRange(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	orgID := claimsFrom(r).OrgID
	resp := reportResponse{
		From: rr.from.Format("2006-01-02"),
		To:   rr.to.AddDate(0, 0, -1).Format("2006-01-02"),
	}

	err := s.db.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM tickets WHERE org_id = $3 AND created_at >= $1 AND created_at < $2),
			(SELECT count(*) FROM tickets WHERE org_id = $3 AND resolved_at >= $1 AND resolved_at < $2),
			(SELECT avg(extract(epoch FROM first_response_at - created_at) / 3600)
			 FROM tickets WHERE org_id = $3 AND first_response_at >= $1 AND first_response_at < $2),
			(SELECT avg(extract(epoch FROM resolved_at - created_at) / 3600)
			 FROM tickets WHERE org_id = $3 AND resolved_at >= $1 AND resolved_at < $2),
			(SELECT avg(CASE WHEN t.first_response_at <= t.created_at + p.first_response_minutes * interval '1 minute'
			                 THEN 1.0 ELSE 0.0 END)
			 FROM tickets t JOIN sla_policies p ON p.org_id = t.org_id AND p.priority = t.priority
			 WHERE t.org_id = $3 AND t.first_response_at >= $1 AND t.first_response_at < $2),
			(SELECT count(*) FROM tickets WHERE org_id = $3 AND satisfaction = 'good' AND rated_at >= $1 AND rated_at < $2),
			(SELECT count(*) FROM tickets WHERE org_id = $3 AND satisfaction = 'bad' AND rated_at >= $1 AND rated_at < $2)`,
		rr.from, rr.to, orgID,
	).Scan(&resp.Created, &resp.Resolved, &resp.AvgFirstResponseHours, &resp.AvgResolutionHours,
		&resp.FirstResponseSLAMet, &resp.Good, &resp.Bad)
	if err != nil {
		internalError(w, err)
		return
	}

	rows, err := s.db.Query(ctx, `
		WITH days AS (
			SELECT generate_series(($1::timestamptz AT TIME ZONE $3)::date,
			                       ($2::timestamptz AT TIME ZONE $3)::date - 1, interval '1 day')::date AS day
		)
		SELECT to_char(d.day, 'YYYY-MM-DD'),
		       (SELECT count(*) FROM tickets WHERE org_id = $4 AND (created_at AT TIME ZONE $3)::date = d.day),
		       (SELECT count(*) FROM tickets WHERE org_id = $4 AND (resolved_at AT TIME ZONE $3)::date = d.day)
		FROM days d ORDER BY d.day`, rr.from, rr.to, rr.tz, orgID)
	if err != nil {
		internalError(w, err)
		return
	}
	resp.Daily, err = pgx.CollectRows(rows, pgx.RowToStructByPos[dailyCount])
	if err != nil {
		internalError(w, err)
		return
	}

	rows, err = s.db.Query(ctx, `
		SELECT u.id, u.name,
		       (SELECT count(*) FROM tickets WHERE assignee_id = u.id AND status NOT IN ('resolved', 'closed')),
		       count(t.id),
		       avg(extract(epoch FROM t.resolved_at - t.created_at) / 3600),
		       count(*) FILTER (WHERE t.satisfaction = 'good'),
		       count(*) FILTER (WHERE t.satisfaction = 'bad')
		FROM users u
		LEFT JOIN tickets t ON t.assignee_id = u.id AND t.resolved_at >= $1 AND t.resolved_at < $2
		WHERE u.org_id = $3 AND u.role IN ('agent', 'admin') AND u.active
		GROUP BY u.id, u.name
		ORDER BY count(t.id) DESC, u.name`, rr.from, rr.to, orgID)
	if err != nil {
		internalError(w, err)
		return
	}
	resp.Agents, err = pgx.CollectRows(rows, pgx.RowToStructByPos[agentReport])
	if err != nil {
		internalError(w, err)
		return
	}

	rows, err = s.db.Query(ctx, `
		SELECT CASE WHEN category = '' THEN 'Sin categoría' ELSE category END, count(*)
		FROM tickets WHERE org_id = $3 AND created_at >= $1 AND created_at < $2
		GROUP BY 1 ORDER BY 2 DESC, 1
		LIMIT 20`, rr.from, rr.to, orgID)
	if err != nil {
		internalError(w, err)
		return
	}
	resp.Categories, err = pgx.CollectRows(rows, pgx.RowToStructByPos[nameCount])
	if err != nil {
		internalError(w, err)
		return
	}
	if resp.Agents == nil {
		resp.Agents = []agentReport{}
	}
	if resp.Categories == nil {
		resp.Categories = []nameCount{}
	}
	writeJSON(w, http.StatusOK, resp)
}

// exportTickets descarga en CSV los tickets creados en el rango (máximo 50 000 filas).
func (s *Server) exportTickets(w http.ResponseWriter, r *http.Request) {
	rr, ok := parseRange(w, r)
	if !ok {
		return
	}
	loc, _ := time.LoadLocation(rr.tz)
	rows, err := s.db.Query(r.Context(), ticketSelect+`
		WHERE t.org_id = $3 AND t.created_at >= $1 AND t.created_at < $2
		ORDER BY t.id
		LIMIT 50000`, rr.from, rr.to, claimsFrom(r).OrgID)
	if err != nil {
		internalError(w, err)
		return
	}
	defer rows.Close()

	filename := fmt.Sprintf("tickets_%s_%s.csv", rr.from.Format("2006-01-02"), rr.to.AddDate(0, 0, -1).Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	// La marca BOM hace que Excel reconozca los acentos.
	w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Título", "Estado", "Prioridad", "Categoría", "Etiquetas", "Solicitante", "Asignado",
		"Creado", "Primera respuesta", "Resuelto", "Valoración"})

	when := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.In(loc).Format("2006-01-02 15:04")
	}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			internalError(w, err)
			return
		}
		assignee := ""
		if t.Assignee != nil {
			assignee = t.Assignee.Name
		}
		rating := map[string]string{"good": "Buena", "bad": "Mala"}[t.Satisfaction]
		cw.Write([]string{
			strconv.FormatInt(t.ID, 10), csvSafe(t.Title), statusText[t.Status], priorityText[t.Priority],
			csvSafe(t.Category), strings.Join(t.Tags, " "), csvSafe(t.Requester.Name), csvSafe(assignee),
			when(&t.CreatedAt), when(t.FirstResponseAt), when(t.ResolvedAt), rating,
		})
	}
	cw.Flush()
}

// csvSafe evita que una hoja de cálculo interprete el texto como fórmula (=, +, -, @).
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
