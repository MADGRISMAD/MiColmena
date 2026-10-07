package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/MADGRISMAD/MiColmena/backend/internal/auth"
)

// platformOrgID es la empresa de la propia plataforma: sus administradores permanentes
// gestionan las demás empresas y reciben las solicitudes de demo.
const platformOrgID int64 = 1

// freePeople es el tamaño de empresa que entra en el plan gratis (1 agente).
const freePeople = 10

type Organization struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	People    int       `json:"people"`
	MaxAgents *int      `json:"max_agents"`
	Suspended bool      `json:"suspended"`
	CreatedAt time.Time `json:"created_at"`
}

const orgColumns = `id, name, slug, people, max_agents, suspended, created_at`

func scanOrg(row pgx.Row) (Organization, error) {
	var o Organization
	err := row.Scan(&o.ID, &o.Name, &o.Slug, &o.People, &o.MaxAgents, &o.Suspended, &o.CreatedAt)
	return o, err
}

// portalOrg busca la empresa de un portal por su slug; vacío = la de la plataforma.
func (s *Server) portalOrg(w http.ResponseWriter, r *http.Request, slug string) (Organization, bool) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	var o Organization
	var err error
	if slug == "" {
		o, err = scanOrg(s.db.QueryRow(r.Context(), `SELECT `+orgColumns+` FROM organizations WHERE id = $1`, platformOrgID))
	} else {
		o, err = scanOrg(s.db.QueryRow(r.Context(), `SELECT `+orgColumns+` FROM organizations WHERE slug = $1`, slug))
	}
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && o.Suspended) {
		writeError(w, http.StatusNotFound, "esa empresa no existe o no está disponible")
		return o, false
	}
	if err != nil {
		internalError(w, err)
		return o, false
	}
	return o, true
}

// agentLimit dice si cabe un agente más en el plan de la empresa ("" = sí). userID es la persona
// que se va a convertir en agente (0 si es nueva), para no contarla dos veces. Bloquea la fila de la
// empresa hasta el final de la transacción para que dos altas simultáneas no se salten el límite.
func agentLimit(ctx context.Context, tx pgx.Tx, orgID, userID int64) (string, error) {
	var max *int
	if err := tx.QueryRow(ctx, `SELECT max_agents FROM organizations WHERE id = $1 FOR UPDATE`, orgID).Scan(&max); err != nil {
		return "", err
	}
	if max == nil {
		return "", nil
	}
	var used int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM users
		WHERE org_id = $1 AND active AND role IN ('agent', 'admin') AND id <> $2`, orgID, userID).Scan(&used); err != nil {
		return "", err
	}
	if used+1 <= *max {
		return "", nil
	}
	noun := "agentes"
	if *max == 1 {
		noun = "agente"
	}
	return fmt.Sprintf("tu plan incluye %d %s; amplía tu plan en «Tu plan» para agregar más", *max, noun), nil
}

var slugInvalid = regexp.MustCompile(`[^a-z0-9]+`)

// slugify: "Ferretería López, S.A." → "ferreteria-lopez-s-a".
func slugify(name string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	plain, _, _ := transform.String(t, strings.ToLower(name))
	slug := strings.Trim(slugInvalid.ReplaceAllString(plain, "-"), "-")
	if len(slug) > 40 {
		slug = strings.Trim(slug[:40], "-")
	}
	return slug
}

// reservedSlugs no pueden usarse como portal para no confundirse con páginas de la plataforma.
var reservedSlugs = map[string]bool{"api": true, "admin": true, "platform": true, "plataforma": true, "help": true, "ayuda": true, "login": true, "signup": true, "soporte": true, "www": true, "behive": true, "micolmena": true}

// signup crea una empresa con su primer administrador e inicia sesión. Empieza en el plan gratis
// (1 agente); para más agentes se pide ampliar el plan.
func (s *Server) signup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Company     string `json:"company"`
		People      int    `json:"people"`
		Name        string `json:"name"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		AcceptTerms bool   `json:"accept_terms"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Company = strings.TrimSpace(in.Company)
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.TrimSpace(in.Email)

	errs := validationErrors{}
	errs.check(in.Company != "", "company", "es obligatorio")
	errs.check(len(in.Company) <= 100, "company", "debe tener como máximo 100 caracteres")
	errs.check(slugify(in.Company) != "", "company", "usa al menos una letra o número")
	errs.check(in.People >= 1 && in.People <= 10_000_000, "people", "indica cuántas personas tiene la empresa")
	checkName(errs, in.Name)
	checkEmail(errs, in.Email)
	checkPassword(errs, "password", in.Password)
	errs.check(in.AcceptTerms, "accept_terms", "debes aceptar los términos y el aviso de privacidad")
	if errs.write(w) {
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		internalError(w, err)
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())

	// El slug se deriva del nombre; si ya existe se le agrega un número.
	base := slugify(in.Company)
	if reservedSlugs[base] {
		base += "-soporte"
	}
	var org Organization
	for i := 1; ; i++ {
		slug := base
		if i > 1 {
			slug = fmt.Sprintf("%s-%d", base, i)
		}
		org, err = scanOrg(tx.QueryRow(r.Context(), `
			INSERT INTO organizations (name, slug, people, max_agents, terms_accepted_at)
			VALUES ($1, $2, $3, 1, now())
			ON CONFLICT (slug) DO NOTHING
			RETURNING `+orgColumns, in.Company, slug, in.People))
		if errors.Is(err, pgx.ErrNoRows) && i < 50 {
			continue
		}
		break
	}
	if err != nil {
		internalError(w, err)
		return
	}
	if err := seedOrg(r.Context(), tx, org.ID); err != nil {
		internalError(w, err)
		return
	}
	u, err := scanUser(tx.QueryRow(r.Context(), `
		INSERT INTO users (name, email, password_hash, role, org_id)
		VALUES ($1, $2, $3, 'admin', $4)
		RETURNING `+userColumns, in.Name, in.Email, hash, org.ID))
	if err != nil {
		internalError(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	s.respondWithToken(w, http.StatusCreated, u)
}

// seedOrg deja lista una empresa nueva: plazos de SLA por defecto y unas categorías de ejemplo.
func seedOrg(ctx context.Context, q querier, orgID int64) error {
	if _, err := q.Exec(ctx, `
		INSERT INTO sla_policies (org_id, priority, first_response_minutes, resolution_minutes) VALUES
			($1, 'urgent', 60, 240), ($1, 'high', 240, 1440), ($1, 'medium', 480, 2880), ($1, 'low', 1440, 5760)`,
		orgID); err != nil {
		return err
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO categories (org_id, name)
		SELECT $1, unnest(ARRAY['Dudas generales', 'Problemas técnicos', 'Facturación'])`, orgID); err != nil {
		return err
	}
	return seedCatalog(ctx, q, orgID)
}

// getPortal da los datos públicos de una empresa para su portal de soporte.
func (s *Server) getPortal(w http.ResponseWriter, r *http.Request) {
	org, ok := s.portalOrg(w, r, r.PathValue("slug"))
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": org.Name, "slug": org.Slug})
}

// orgUsage es la empresa del usuario con lo que usa de su plan.
type orgUsage struct {
	Organization
	Agents int `json:"agents"`
	// Platform: la empresa de la plataforma (sin plan ni límites).
	Platform bool `json:"platform"`
}

func (s *Server) loadUsage(ctx context.Context, orgID int64) (orgUsage, error) {
	var u orgUsage
	o, err := scanOrg(s.db.QueryRow(ctx, `SELECT `+orgColumns+` FROM organizations WHERE id = $1`, orgID))
	if err != nil {
		return u, err
	}
	u.Organization = o
	u.Platform = o.ID == platformOrgID
	err = s.db.QueryRow(ctx, `
		SELECT count(*) FROM users WHERE org_id = $1 AND active AND role IN ('agent', 'admin')`, orgID).Scan(&u.Agents)
	return u, err
}

func (s *Server) getOrg(w http.ResponseWriter, r *http.Request) {
	u, err := s.loadUsage(r.Context(), claimsFrom(r).OrgID)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// updateOrg cambia el nombre de la empresa (el portal conserva su dirección).
func (s *Server) updateOrg(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	errs := validationErrors{}
	errs.check(in.Name != "", "name", "es obligatorio")
	errs.check(len(in.Name) <= 100, "name", "debe tener como máximo 100 caracteres")
	if errs.write(w) {
		return
	}
	orgID := claimsFrom(r).OrgID
	if _, err := s.db.Exec(r.Context(), `UPDATE organizations SET name = $2 WHERE id = $1`, orgID, in.Name); err != nil {
		internalError(w, err)
		return
	}
	s.getOrg(w, r)
}

// requestUpgrade pide ampliar el plan: llega como solicitud a la plataforma con los datos de la empresa.
func (s *Server) requestUpgrade(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Agents  int    `json:"agents"`
		People  int    `json:"people"`
		Message string `json:"message"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Message = strings.TrimSpace(in.Message)
	errs := validationErrors{}
	errs.check(in.Agents >= 1 && in.Agents <= 100_000, "agents", "indica cuántos agentes necesitas")
	errs.check(in.People >= 1 && in.People <= 10_000_000, "people", "indica cuántas personas tiene la empresa")
	errs.check(len(in.Message) <= 2000, "message", "es demasiado largo")
	if errs.write(w) {
		return
	}
	claims := claimsFrom(r)
	var name, email, company string
	if err := s.db.QueryRow(r.Context(), `
		SELECT u.name, u.email, o.name FROM users u JOIN organizations o ON o.id = u.org_id WHERE u.id = $1`,
		claims.UserID()).Scan(&name, &email, &company); err != nil {
		internalError(w, err)
		return
	}
	if err := s.saveLead(r.Context(), leadInput{
		Name: name, Company: company, Email: email, Agents: in.Agents, People: in.People,
		Message: in.Message, OrgID: &claims.OrgID,
	}); err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// --- Panel de la plataforma: solo administradores permanentes de la empresa de la plataforma ---

func (s *Server) platformOnly(next http.HandlerFunc) http.Handler {
	return s.authed(func(w http.ResponseWriter, r *http.Request) {
		c := claimsFrom(r)
		var u User
		var err error
		if c.OrgID == platformOrgID && c.Role == auth.RoleAdmin {
			u, err = scanUser(s.db.QueryRow(r.Context(), `SELECT `+userColumns+` FROM users WHERE id = $1`, c.UserID()))
		}
		if err != nil {
			internalError(w, err)
			return
		}
		if !u.Permanent {
			writeError(w, http.StatusForbidden, "solo para los administradores de la plataforma")
			return
		}
		next(w, r)
	})
}

type platformOrg struct {
	Organization
	Agents     int        `json:"agents"`
	Customers  int        `json:"customers"`
	Tickets    int        `json:"tickets"`
	LastTicket *time.Time `json:"last_ticket_at"`
}

func (s *Server) listPlatformOrgs(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `
		SELECT `+prefixed("o.", orgColumns)+`,
		       (SELECT count(*) FROM users WHERE org_id = o.id AND active AND role IN ('agent', 'admin')),
		       (SELECT count(*) FROM users WHERE org_id = o.id AND role = 'customer'),
		       (SELECT count(*) FROM tickets WHERE org_id = o.id),
		       (SELECT max(created_at) FROM tickets WHERE org_id = o.id)
		FROM organizations o
		ORDER BY o.id DESC
		LIMIT 1000`)
	if err != nil {
		internalError(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (platformOrg, error) {
		var p platformOrg
		o := &p.Organization
		err := row.Scan(&o.ID, &o.Name, &o.Slug, &o.People, &o.MaxAgents, &o.Suspended, &o.CreatedAt,
			&p.Agents, &p.Customers, &p.Tickets, &p.LastTicket)
		return p, err
	})
	if err != nil {
		internalError(w, err)
		return
	}
	if items == nil {
		items = []platformOrg{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// updatePlatformOrg ajusta el plan de una empresa (agentes y personas) o la suspende.
func (s *Server) updatePlatformOrg(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		People    *int  `json:"people"`
		MaxAgents *int  `json:"max_agents"`
		Unlimited bool  `json:"unlimited"`
		Suspended *bool `json:"suspended"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	errs := validationErrors{}
	errs.check(in.People == nil || (*in.People >= 1 && *in.People <= 10_000_000), "people", "valor inválido")
	errs.check(in.MaxAgents == nil || (*in.MaxAgents >= 1 && *in.MaxAgents <= 100_000), "max_agents", "valor inválido")
	errs.check(id != platformOrgID || in.Suspended == nil || !*in.Suspended, "suspended", "la empresa de la plataforma no se puede suspender")
	if errs.write(w) {
		return
	}
	o, err := scanOrg(s.db.QueryRow(r.Context(), `
		UPDATE organizations SET
			people     = coalesce($2, people),
			max_agents = CASE WHEN $4 THEN NULL ELSE coalesce($3, max_agents) END,
			suspended  = coalesce($5, suspended)
		WHERE id = $1
		RETURNING `+orgColumns, id, in.People, in.MaxAgents, in.Unlimited, in.Suspended))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "empresa no encontrada")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}
