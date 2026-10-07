package mail_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/MADGRISMAD/MiColmena/backend/internal/mail"
	"github.com/MADGRISMAD/MiColmena/backend/internal/testutil"
)

type fakeSender struct {
	mu   sync.Mutex
	sent []mail.Message
	fail map[string]bool
}

func (f *fakeSender) Send(_ context.Context, m mail.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail[m.To] {
		return errors.New("buzón lleno")
	}
	f.sent = append(f.sent, m)
	return nil
}

func TestWorkerSendsAndRetries(t *testing.T) {
	ctx := context.Background()
	pool := testutil.NewPool(t)
	for _, to := range []string{"a@x.test", "b@x.test"} {
		if _, err := pool.Exec(ctx, `INSERT INTO email_outbox (to_email, subject, body) VALUES ($1, 'Hola', 'Texto')`, to); err != nil {
			t.Fatal(err)
		}
	}
	sender := &fakeSender{fail: map[string]bool{"b@x.test": true}}
	w := mail.Worker{DB: pool, Sender: sender}

	n, err := w.SendBatch(ctx)
	if err != nil || n != 2 {
		t.Fatalf("SendBatch = %d, %v", n, err)
	}
	if len(sender.sent) != 1 || sender.sent[0].To != "a@x.test" {
		t.Errorf("enviados = %+v", sender.sent)
	}

	var sent, attempts int
	var lastError string
	pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE sent_at IS NOT NULL) FROM email_outbox`).Scan(&sent)
	pool.QueryRow(ctx, `SELECT attempts, last_error FROM email_outbox WHERE to_email = 'b@x.test'`).Scan(&attempts, &lastError)
	if sent != 1 || attempts != 1 || lastError != "buzón lleno" {
		t.Errorf("sent=%d attempts=%d last_error=%q", sent, attempts, lastError)
	}

	// El fallido espera antes de reintentar: no se vuelve a tomar enseguida.
	if n, _ := w.SendBatch(ctx); n != 0 {
		t.Errorf("el reintento debía esperar; se tomaron %d", n)
	}
	pool.Exec(ctx, `UPDATE email_outbox SET send_after = now() WHERE sent_at IS NULL`)
	sender.fail = nil
	if n, _ := w.SendBatch(ctx); n != 1 || len(sender.sent) != 2 {
		t.Errorf("reintento: n=%d enviados=%d", n, len(sender.sent))
	}
}

func TestComposeEncodesSubjectAndBlocksHeaderInjection(t *testing.T) {
	msg := string(mail.Compose("BeHIve <no-reply@x.test>", mail.Message{
		To: "a@x.test", Subject: "Ticket resuelto\r\nBcc: espía@x.test", Body: "Línea 1\nLínea 2",
	}))
	if strings.Contains(msg, "\r\nBcc:") {
		t.Errorf("se inyectó una cabecera:\n%s", msg)
	}
	if !strings.Contains(msg, "Subject: =?utf-8?q?") {
		t.Errorf("el asunto debe ir codificado:\n%s", msg)
	}
	if !strings.Contains(msg, "Línea 1\r\nLínea 2") {
		t.Errorf("el cuerpo debe usar CRLF:\n%s", msg)
	}
}
