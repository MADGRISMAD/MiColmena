package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Lead es una solicitud de demo enviada desde la landing.
type Lead struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Company  string `json:"company"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	TeamSize string `json:"team_size"`
	Plan     string `json:"plan"`
	Message  string `json:"message"`
	// Lo elegido en la calculadora de precios; 0 = no lo indicó.
	Agents int `json:"agents"`
	People int `json:"people"`
	// OrgID: si la pidió una empresa que ya usa BeHIve (ampliar plan).
	OrgID     *int64    `json:"org_id"`
	Handled   bool      `json:"handled"`
	CreatedAt time.Time `json:"created_at"`
}

const leadColumns = `id, name, company, email, phone, team_size, plan, message, agents, people, org_id, handled, created_at`

var (
	leadTeamSizes = []string{"", "1-3", "4-10", "11-25", "26+"}
	leadPlans     = []string{"", "emprendedor", "profesional", "empresa"}
)

// createLead guarda la solicitud y avisa a los administradores (campana y correo).
// Es pública, por eso pasa por el límite de intentos por IP.
func (s *Server) createLead(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		Company  string `json:"company"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		TeamSize string `json:"team_size"`
		Plan     string `json:"plan"`
		Message  string `json:"message"`
		Agents   int    `json:"agents"`
		People   int    `json:"people"`
		// Campo trampa: invisible para personas; los bots lo rellenan.
		Website string `json:"website"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Company = strings.TrimSpace(in.Company)
	in.Email = strings.TrimSpace(in.Email)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Message = strings.TrimSpace(in.Message)

	errs := validationErrors{}
	checkName(errs, in.Name)
	errs.check(in.Company != "", "company", "es obligatorio")
	errs.check(len(in.Company) <= 150, "company", "debe tener como máximo 150 caracteres")
	checkEmail(errs, in.Email)
	errs.check(len(in.Phone) <= 30, "phone", "debe tener como máximo 30 caracteres")
	errs.check(slices.Contains(leadTeamSizes, in.TeamSize), "team_size", "valor inválido")
	errs.check(slices.Contains(leadPlans, in.Plan), "plan", "valor inválido")
	errs.check(len(in.Message) <= 2000, "message", "es demasiado largo")
	errs.check(in.Agents >= 0 && in.Agents <= 100_000, "agents", "valor inválido")
	errs.check(in.People >= 0 && in.People <= 10_000_000, "people", "valor inválido")
	if errs.write(w) {
		return
	}
	if in.Website != "" {
		// Se responde como si todo hubiera ido bien para no dar pistas al bot.
		w.WriteHeader(http.StatusCreated)
		return
	}

	if err := s.saveLead(r.Context(), leadInput{
		Name: in.Name, Company: in.Company, Email: in.Email, Phone: in.Phone, TeamSize: in.TeamSize,
		Plan: in.Plan, Message: in.Message, Agents: in.Agents, People: in.People,
	}); err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

type leadInput struct {
	Name, Company, Email, Phone, TeamSize, Plan, Message string
	Agents, People                                       int
	// OrgID: la empresa cliente que pide ampliar su plan (nil si viene de la landing).
	OrgID *int64
}

// saveLead guarda la solicitud y avisa a los administradores de la plataforma (campana y correo).
func (s *Server) saveLead(ctx context.Context, in leadInput) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO leads (name, company, email, phone, team_size, plan, message, agents, people, org_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		in.Name, in.Company, in.Email, in.Phone, in.TeamSize, in.Plan, in.Message, in.Agents, in.People, in.OrgID); err != nil {
		return err
	}

	kind, subject := "pidió una demo", "Nueva solicitud de demo: "
	if in.OrgID != nil {
		kind, subject = "pidió ampliar su plan", "Ampliar plan: "
	}
	summary := fmt.Sprintf("%s (%s) %s", in.Name, in.Company, kind)
	body := fmt.Sprintf("%s\n\nNombre: %s\nEmpresa: %s\nEmail: %s\nTeléfono: %s\nAgentes: %s\nPersonas en la empresa: %s\n\n%s\n\nVer solicitudes: %s/admin/leads",
		summary, in.Name, in.Company, in.Email, orDash(in.Phone), countOrDash(in.Agents), countOrDash(in.People), in.Message, s.appURL)
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, kind, summary)
		SELECT id, 'lead', $1 FROM users WHERE role = 'admin' AND active AND org_id = $2`,
		excerpt(summary, 200), platformOrgID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO email_outbox (to_email, subject, body)
		SELECT email, $1, $2 FROM users WHERE role = 'admin' AND active AND email_notifications AND org_id = $3`,
		subject+excerpt(in.Company, 80), body, platformOrgID); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT id FROM users WHERE role = 'admin' AND active AND org_id = $1`, platformOrgID)
	if err != nil {
		return err
	}
	admins, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.broker.publish(fanout{users: admins})
	return nil
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func countOrDash(n int) string {
	if n == 0 {
		return "—"
	}
	return strconv.Itoa(n)
}

func (s *Server) listLeads(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT `+leadColumns+` FROM leads ORDER BY handled, id DESC LIMIT 500`)
	if err != nil {
		internalError(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Lead])
	if err != nil {
		internalError(w, err)
		return
	}
	if items == nil {
		items = []Lead{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// updateLead marca la solicitud como atendida (o pendiente otra vez).
func (s *Server) updateLead(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Handled bool `json:"handled"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	rows, err := s.db.Query(r.Context(), `UPDATE leads SET handled = $2 WHERE id = $1 RETURNING `+leadColumns, id, in.Handled)
	if err != nil {
		internalError(w, err)
		return
	}
	lead, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Lead])
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "solicitud no encontrada")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, lead)
}
