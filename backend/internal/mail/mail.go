// Package mail envía los correos que la API deja en la tabla email_outbox.
package mail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Message es un correo de texto plano.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender entrega un correo.
type Sender interface {
	Send(ctx context.Context, m Message) error
}

// LogSender no envía nada: escribe el correo en el log. Es el modo por defecto mientras
// no haya un servidor SMTP configurado, útil en desarrollo para ver enlaces de recuperación.
type LogSender struct{}

func (LogSender) Send(_ context.Context, m Message) error {
	slog.Info("correo (sin SMTP configurado, solo se registra)", "to", m.To, "subject", m.Subject, "body", m.Body)
	return nil
}

// SMTPSender envía por SMTP. Con el puerto 465 usa TLS directo; con otro, STARTTLS si el servidor lo ofrece.
type SMTPSender struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (s SMTPSender) Send(ctx context.Context, m Message) error {
	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	var conn net.Conn
	var err error
	if s.Port == 465 {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: s.Host})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("conectar a %s: %w", addr, err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	} else {
		conn.SetDeadline(time.Now().Add(30 * time.Second))
	}
	c, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()

	if s.Port != 465 {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: s.Host}); err != nil {
				return fmt.Errorf("STARTTLS: %w", err)
			}
		}
	}
	if s.Username != "" {
		if err := c.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
			return fmt.Errorf("autenticación SMTP: %w", err)
		}
	}
	if err := c.Mail(addressOf(s.From)); err != nil {
		return err
	}
	if err := c.Rcpt(m.To); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(Compose(s.From, m)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// addressOf extrae el email de "Nombre <email>".
func addressOf(from string) string {
	if i := strings.LastIndex(from, "<"); i >= 0 {
		return strings.TrimSuffix(from[i+1:], ">")
	}
	return from
}

// Compose arma el mensaje con cabeceras. El asunto se codifica para admitir acentos.
func Compose(from string, m Message) []byte {
	var b strings.Builder
	header := func(k, v string) { b.WriteString(k + ": " + v + "\r\n") }
	header("From", from)
	header("To", m.To)
	header("Subject", mime.QEncoding.Encode("utf-8", stripNewlines(m.Subject)))
	header("Date", time.Now().Format(time.RFC1123Z))
	header("MIME-Version", "1.0")
	header("Content-Type", "text/plain; charset=utf-8")
	header("Content-Transfer-Encoding", "8bit")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(strings.ReplaceAll(m.Body, "\r\n", "\n"), "\n", "\r\n"))
	return []byte(b.String())
}

// stripNewlines evita que un título con saltos de línea inyecte cabeceras.
func stripNewlines(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
}

const maxAttempts = 6

// Worker envía los correos pendientes de email_outbox. Varias instancias pueden correr a la vez:
// cada correo se reserva con FOR UPDATE SKIP LOCKED.
type Worker struct {
	DB       *pgxpool.Pool
	Sender   Sender
	Interval time.Duration
}

// Run procesa la cola hasta que ctx se cancela.
func (w Worker) Run(ctx context.Context) {
	interval := w.Interval
	if interval == 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		for {
			n, err := w.SendBatch(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("cola de correo", "error", err)
			}
			if n == 0 || err != nil {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// SendBatch envía hasta 10 correos pendientes y devuelve cuántos intentó.
func (w Worker) SendBatch(ctx context.Context) (int, error) {
	tx, err := w.DB.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id, to_email, subject, body, attempts FROM email_outbox
		WHERE sent_at IS NULL AND send_after <= now() AND attempts < $1
		ORDER BY id
		LIMIT 10
		FOR UPDATE SKIP LOCKED`, maxAttempts)
	if err != nil {
		return 0, err
	}
	type pending struct {
		id       int64
		msg      Message
		attempts int
	}
	batch, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (pending, error) {
		var p pending
		err := row.Scan(&p.id, &p.msg.To, &p.msg.Subject, &p.msg.Body, &p.attempts)
		return p, err
	})
	if err != nil {
		return 0, err
	}

	for _, p := range batch {
		sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := w.Sender.Send(sendCtx, p.msg)
		cancel()
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE email_outbox SET sent_at = now(), attempts = attempts + 1 WHERE id = $1`, p.id)
			if err != nil {
				return 0, err
			}
			continue
		}
		// Reintento con espera creciente: 1, 4, 9, 16, 25 minutos.
		backoffMinutes := (p.attempts + 1) * (p.attempts + 1)
		slog.Warn("no se pudo enviar el correo", "id", p.id, "attempt", p.attempts+1, "error", err)
		if _, err := tx.Exec(ctx, `
			UPDATE email_outbox SET attempts = attempts + 1, last_error = $2,
			       send_after = now() + make_interval(mins => $3)
			WHERE id = $1`, p.id, err.Error(), backoffMinutes); err != nil {
			return 0, err
		}
	}
	return len(batch), tx.Commit(ctx)
}
