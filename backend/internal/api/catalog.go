package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Catálogo de servicios: categorías con servicios que se piden con un formulario. Pedir uno crea un
// ticket de tipo «solicitud» con las respuestas del formulario.

// CatalogRef es el servicio del que viene un ticket.
type CatalogRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ServiceCategory struct {
	ID          int64         `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Icon        string        `json:"icon"`
	Position    int           `json:"position"`
	Items       []ServiceItem `json:"items"`
}

type ServiceItem struct {
	ID             int64       `json:"id"`
	CategoryID     int64       `json:"category_id"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	Fields         []FormField `json:"fields"`
	Priority       string      `json:"priority"`
	TicketCategory string      `json:"ticket_category"`
	Active         bool        `json:"active"`
	Position       int         `json:"position"`
	CreatedAt      time.Time   `json:"created_at"`
}

// FormField es una pregunta del formulario de un servicio.
type FormField struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"` // text, textarea, select, number, date
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
}

var (
	fieldTypes     = []string{"text", "textarea", "select", "number", "date"}
	fieldKeyRegexp = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)
	// Iconos disponibles para las categorías (los dibuja el frontend).
	catalogIcons = []string{"package", "shield", "mail", "phone", "users", "wrench", "laptop", "key", "chart"}
)

const (
	maxFormFields = 20
	maxFieldValue = 2000
)

const itemColumns = `id, category_id, name, description, fields, priority, ticket_category, active, position, created_at`

func scanItem(row pgx.Row) (ServiceItem, error) {
	var it ServiceItem
	var raw []byte
	err := row.Scan(&it.ID, &it.CategoryID, &it.Name, &it.Description, &raw, &it.Priority,
		&it.TicketCategory, &it.Active, &it.Position, &it.CreatedAt)
	if err != nil {
		return it, err
	}
	if err := json.Unmarshal(raw, &it.Fields); err != nil || it.Fields == nil {
		it.Fields = []FormField{}
	}
	return it, nil
}

// getCatalog devuelve las categorías con sus servicios. Quien no es del equipo solo ve los activos;
// los administradores pueden pedir todo con ?all=1 para editarlo.
func (s *Server) getCatalog(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	all := r.URL.Query().Get("all") == "1" && claims.Role == "admin"

	rows, err := s.db.Query(r.Context(), `
		SELECT id, name, description, icon, position FROM service_categories
		WHERE org_id = $1 ORDER BY position, lower(name)`, claims.OrgID)
	if err != nil {
		internalError(w, err)
		return
	}
	cats, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ServiceCategory, error) {
		var c ServiceCategory
		err := row.Scan(&c.ID, &c.Name, &c.Description, &c.Icon, &c.Position)
		c.Items = []ServiceItem{}
		return c, err
	})
	if err != nil {
		internalError(w, err)
		return
	}

	query := `SELECT ` + itemColumns + ` FROM service_items WHERE org_id = $1`
	if !all {
		query += ` AND active`
	}
	rows, err = s.db.Query(r.Context(), query+` ORDER BY position, lower(name)`, claims.OrgID)
	if err != nil {
		internalError(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ServiceItem, error) { return scanItem(row) })
	if err != nil {
		internalError(w, err)
		return
	}
	byID := map[int64]*ServiceCategory{}
	for i := range cats {
		byID[cats[i].ID] = &cats[i]
	}
	for _, it := range items {
		if c := byID[it.CategoryID]; c != nil {
			c.Items = append(c.Items, it)
		}
	}
	if cats == nil {
		cats = []ServiceCategory{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": cats})
}

// --- Administración ---

type categoryInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Position    int    `json:"position"`
}

func readCategory(w http.ResponseWriter, r *http.Request) (categoryInput, bool) {
	var in categoryInput
	if !decodeJSON(w, r, &in) {
		return in, false
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if in.Icon == "" {
		in.Icon = "package"
	}
	errs := validationErrors{}
	errs.check(in.Name != "", "name", "es obligatorio")
	errs.check(len(in.Name) <= 80, "name", "debe tener como máximo 80 caracteres")
	errs.check(len(in.Description) <= 300, "description", "debe tener como máximo 300 caracteres")
	errs.check(slices.Contains(catalogIcons, in.Icon), "icon", "valor inválido")
	errs.check(in.Position >= 0 && in.Position < 10000, "position", "valor inválido")
	return in, !errs.write(w)
}

func isUnique(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

func (s *Server) createServiceCategory(w http.ResponseWriter, r *http.Request) {
	in, ok := readCategory(w, r)
	if !ok {
		return
	}
	var c ServiceCategory
	err := s.db.QueryRow(r.Context(), `
		INSERT INTO service_categories (org_id, name, description, icon, position)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, description, icon, position`,
		claimsFrom(r).OrgID, in.Name, in.Description, in.Icon, in.Position,
	).Scan(&c.ID, &c.Name, &c.Description, &c.Icon, &c.Position)
	if isUnique(err) {
		validationErrors{"name": "ya existe una categoría con ese nombre"}.write(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	c.Items = []ServiceItem{}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) updateServiceCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	in, ok := readCategory(w, r)
	if !ok {
		return
	}
	var c ServiceCategory
	err := s.db.QueryRow(r.Context(), `
		UPDATE service_categories SET name = $3, description = $4, icon = $5, position = $6
		WHERE id = $1 AND org_id = $2
		RETURNING id, name, description, icon, position`,
		id, claimsFrom(r).OrgID, in.Name, in.Description, in.Icon, in.Position,
	).Scan(&c.ID, &c.Name, &c.Description, &c.Icon, &c.Position)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "categoría no encontrada")
		return
	}
	if isUnique(err) {
		validationErrors{"name": "ya existe una categoría con ese nombre"}.write(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	c.Items = []ServiceItem{}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) deleteServiceCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	tag, err := s.db.Exec(r.Context(), `DELETE FROM service_categories WHERE id = $1 AND org_id = $2`,
		id, claimsFrom(r).OrgID)
	if err != nil {
		internalError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "categoría no encontrada")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type itemInput struct {
	CategoryID     int64       `json:"category_id"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	Fields         []FormField `json:"fields"`
	Priority       string      `json:"priority"`
	TicketCategory string      `json:"ticket_category"`
	Active         *bool       `json:"active"`
	Position       int         `json:"position"`
}

func (s *Server) readItem(w http.ResponseWriter, r *http.Request) (itemInput, bool) {
	claims := claimsFrom(r)
	var in itemInput
	if !decodeJSON(w, r, &in) {
		return in, false
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if in.Priority == "" {
		in.Priority = "medium"
	}
	if in.Fields == nil {
		in.Fields = []FormField{}
	}
	errs := validationErrors{}
	errs.check(in.Name != "", "name", "es obligatorio")
	errs.check(len(in.Name) <= 120, "name", "debe tener como máximo 120 caracteres")
	errs.check(len(in.Description) <= 1000, "description", "debe tener como máximo 1000 caracteres")
	errs.check(slices.Contains(ticketPriorities, in.Priority), "priority", "valor inválido")
	errs.check(in.Position >= 0 && in.Position < 10000, "position", "valor inválido")
	errs.check(len(in.Fields) <= maxFormFields, "fields", fmt.Sprintf("máximo %d campos", maxFormFields))
	seen := map[string]bool{}
	for i := range in.Fields {
		f := &in.Fields[i]
		f.Label = strings.TrimSpace(f.Label)
		f.Key = strings.TrimSpace(f.Key)
		if f.Key == "" {
			f.Key = slugKey(f.Label, i)
		}
		errs.check(f.Label != "" && len(f.Label) <= 100, "fields", "cada campo necesita un nombre de hasta 100 caracteres")
		errs.check(fieldKeyRegexp.MatchString(f.Key), "fields", "identificador de campo inválido")
		errs.check(!seen[f.Key], "fields", "hay dos campos con el mismo identificador")
		seen[f.Key] = true
		errs.check(slices.Contains(fieldTypes, f.Type), "fields", "tipo de campo inválido")
		if f.Type == "select" {
			errs.check(len(f.Options) >= 1 && len(f.Options) <= 30, "fields", "una lista necesita entre 1 y 30 opciones")
			for j := range f.Options {
				f.Options[j] = strings.TrimSpace(f.Options[j])
				errs.check(f.Options[j] != "" && len(f.Options[j]) <= 100, "fields", "opción inválida")
			}
		} else {
			f.Options = nil
		}
	}
	if errs.write(w) {
		return in, false
	}
	cat, err := s.canonicalCategory(r.Context(), claims.OrgID, in.TicketCategory)
	if err != nil {
		internalError(w, err)
		return in, false
	}
	if cat == nil {
		validationErrors{"ticket_category": "no es una categoría de tickets válida"}.write(w)
		return in, false
	}
	in.TicketCategory = *cat
	var exists bool
	if err := s.db.QueryRow(r.Context(),
		`SELECT EXISTS (SELECT 1 FROM service_categories WHERE id = $1 AND org_id = $2)`,
		in.CategoryID, claims.OrgID).Scan(&exists); err != nil {
		internalError(w, err)
		return in, false
	}
	if !exists {
		validationErrors{"category_id": "la categoría no existe"}.write(w)
		return in, false
	}
	return in, true
}

var nonKey = regexp.MustCompile(`[^a-z0-9]+`)

// slugKey genera el identificador de un campo a partir de su nombre.
func slugKey(label string, i int) string {
	k := strings.Trim(nonKey.ReplaceAllString(strings.ToLower(slugify(label)), "_"), "_")
	if k == "" || k[0] < 'a' || k[0] > 'z' {
		k = "campo_" + k
	}
	k = strings.TrimRight(k, "_")
	if len(k) > 34 {
		k = k[:34]
	}
	return k + "_" + strconv.Itoa(i+1)
}

func (s *Server) createServiceItem(w http.ResponseWriter, r *http.Request) {
	in, ok := s.readItem(w, r)
	if !ok {
		return
	}
	fields, _ := json.Marshal(in.Fields)
	active := in.Active == nil || *in.Active
	it, err := scanItem(s.db.QueryRow(r.Context(), `
		INSERT INTO service_items (org_id, category_id, name, description, fields, priority, ticket_category, active, position)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING `+itemColumns,
		claimsFrom(r).OrgID, in.CategoryID, in.Name, in.Description, fields, in.Priority, in.TicketCategory, active, in.Position))
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, it)
}

func (s *Server) updateServiceItem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	in, ok := s.readItem(w, r)
	if !ok {
		return
	}
	fields, _ := json.Marshal(in.Fields)
	it, err := scanItem(s.db.QueryRow(r.Context(), `
		UPDATE service_items SET category_id = $3, name = $4, description = $5, fields = $6, priority = $7,
			ticket_category = $8, active = coalesce($9, active), position = $10
		WHERE id = $1 AND org_id = $2
		RETURNING `+itemColumns,
		id, claimsFrom(r).OrgID, in.CategoryID, in.Name, in.Description, fields, in.Priority, in.TicketCategory,
		in.Active, in.Position))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "servicio no encontrado")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, it)
}

func (s *Server) deleteServiceItem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	tag, err := s.db.Exec(r.Context(), `DELETE FROM service_items WHERE id = $1 AND org_id = $2`,
		id, claimsFrom(r).OrgID)
	if err != nil {
		internalError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "servicio no encontrado")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Pedir un servicio ---

// requestService crea un ticket de tipo «solicitud» con las respuestas del formulario.
func (s *Server) requestService(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Values  map[string]any `json:"values"`
		Notes   string         `json:"notes"`
		AssetID *int64         `json:"asset_id"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Notes = strings.TrimSpace(in.Notes)

	it, err := scanItem(s.db.QueryRow(r.Context(),
		`SELECT `+itemColumns+` FROM service_items WHERE id = $1 AND org_id = $2 AND active`, id, claims.OrgID))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "servicio no encontrado")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	errs := validationErrors{}
	errs.check(len(in.Notes) <= 5000, "notes", "es demasiado largo")
	values := map[string]string{}
	var lines []string
	for _, f := range it.Fields {
		raw := in.Values[f.Key]
		v := ""
		switch x := raw.(type) {
		case nil:
		case string:
			v = strings.TrimSpace(x)
		case float64:
			v = strconv.FormatFloat(x, 'f', -1, 64)
		default:
			errs["values."+f.Key] = "valor inválido"
			continue
		}
		field := "values." + f.Key
		if v == "" {
			errs.check(!f.Required, field, "es obligatorio")
			continue
		}
		errs.check(len(v) <= maxFieldValue, field, "es demasiado largo")
		switch f.Type {
		case "select":
			errs.check(slices.Contains(f.Options, v), field, "elige una de las opciones")
		case "number":
			_, err := strconv.ParseFloat(v, 64)
			errs.check(err == nil, field, "debe ser un número")
		case "date":
			_, err := time.Parse("2006-01-02", v)
			errs.check(err == nil, field, "debe ser una fecha válida")
		}
		values[f.Key] = v
		lines = append(lines, f.Label+": "+v)
	}
	if in.AssetID != nil {
		ok, err := s.assetUsable(r.Context(), claims, claims.UserID(), *in.AssetID)
		if err != nil {
			internalError(w, err)
			return
		}
		errs.check(ok, "asset_id", "el equipo no existe")
	}
	if errs.write(w) {
		return
	}
	if in.Notes != "" {
		lines = append(lines, "", in.Notes)
	}
	custom, _ := json.Marshal(values)

	category := it.TicketCategory
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	t, out, err := s.insertTicket(r.Context(), tx, newTicket{
		Title: it.Name, Description: strings.Join(lines, "\n"), Priority: it.Priority, Category: category,
		CustomFields: custom, Kind: "request", CatalogItemID: &it.ID, AssetID: in.AssetID,
		RequesterID: claims.UserID(), ActorID: claims.UserID(), OrgID: claims.OrgID,
	})
	if err != nil {
		internalError(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	s.broker.publish(out)
	writeJSON(w, http.StatusCreated, t)
}

// seedCatalog deja unos servicios de ejemplo en una empresa nueva para que el catálogo no esté vacío.
func seedCatalog(ctx context.Context, q querier, orgID int64) error {
	type item struct {
		name, desc, priority string
		fields               []FormField
	}
	type cat struct {
		name, desc, icon string
		items            []item
	}
	catalog := []cat{
		{"Accesos y seguridad", "Pide acceso a sistemas, carpetas o aplicaciones.", "key", []item{
			{"Acceso a un sistema", "Pide acceso a una aplicación o carpeta compartida.", "medium", []FormField{
				{Key: "sistema", Label: "¿A qué sistema o carpeta?", Type: "text", Required: true},
				{Key: "nivel", Label: "Nivel de acceso", Type: "select", Required: true, Options: []string{"Solo lectura", "Lectura y escritura", "Administrador"}},
				{Key: "motivo", Label: "Motivo", Type: "textarea"},
			}},
			{"Restablecer contraseña", "Recupera el acceso a tu cuenta.", "high", []FormField{
				{Key: "cuenta", Label: "¿De qué cuenta o sistema?", Type: "text", Required: true},
			}},
		}},
		{"Equipo y dispositivos", "Pide un equipo nuevo, un accesorio o una reparación.", "laptop", []item{
			{"Equipo de cómputo nuevo", "Laptop o computadora de escritorio.", "medium", []FormField{
				{Key: "tipo", Label: "Tipo de equipo", Type: "select", Required: true, Options: []string{"Laptop", "Escritorio"}},
				{Key: "para_quien", Label: "¿Para quién es?", Type: "text", Required: true},
				{Key: "necesito_antes", Label: "Lo necesito antes del", Type: "date"},
			}},
			{"Accesorio o periférico", "Mouse, teclado, monitor, diadema…", "low", []FormField{
				{Key: "accesorio", Label: "¿Qué necesitas?", Type: "text", Required: true},
				{Key: "cantidad", Label: "Cantidad", Type: "number"},
			}},
		}},
		{"Altas y bajas de personal", "Prepara todo para quien entra o sale de la empresa.", "users", []item{
			{"Alta de nuevo empleado", "Cuentas, equipo y accesos para una persona nueva.", "medium", []FormField{
				{Key: "nombre", Label: "Nombre de la persona", Type: "text", Required: true},
				{Key: "puesto", Label: "Puesto", Type: "text", Required: true},
				{Key: "fecha_ingreso", Label: "Fecha de ingreso", Type: "date", Required: true},
			}},
			{"Baja de empleado", "Desactiva cuentas y recupera el equipo.", "high", []FormField{
				{Key: "nombre", Label: "Nombre de la persona", Type: "text", Required: true},
				{Key: "ultimo_dia", Label: "Último día", Type: "date", Required: true},
			}},
		}},
	}
	for ci, c := range catalog {
		var catID int64
		if err := q.QueryRow(ctx, `
			INSERT INTO service_categories (org_id, name, description, icon, position)
			VALUES ($1, $2, $3, $4, $5) RETURNING id`, orgID, c.name, c.desc, c.icon, ci).Scan(&catID); err != nil {
			return err
		}
		for ii, it := range c.items {
			fields, _ := json.Marshal(it.fields)
			if _, err := q.Exec(ctx, `
				INSERT INTO service_items (org_id, category_id, name, description, fields, priority, position)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`, orgID, catID, it.name, it.desc, fields, it.priority, ii); err != nil {
				return err
			}
		}
	}
	return nil
}
