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

	"github.com/MADGRISMAD/MiColmena/backend/internal/auth"
)

// Activos: el inventario de equipos de la empresa (computadoras, teléfonos, redes…) con su
// responsable, estado y garantía. Se vinculan a tickets para ver el historial de cada equipo.

var (
	assetCategories = []string{"computer", "phone", "monitor", "network", "peripheral", "software", "other"}
	assetStates     = []string{"in_stock", "in_use", "in_repair", "retired"}
)

// warrantyWarnDays es la anticipación con la que se avisa que una garantía está por vencer.
const warrantyWarnDays = 60

type Asset struct {
	ID            int64     `json:"id"`
	Tag           string    `json:"tag"`
	Name          string    `json:"name"`
	Category      string    `json:"category"`
	Model         string    `json:"model"`
	Serial        string    `json:"serial"`
	State         string    `json:"state"`
	AssignedTo    *UserRef  `json:"assigned_to"`
	Location      string    `json:"location"`
	PurchaseDate  *string   `json:"purchase_date"`
	PurchaseCost  *float64  `json:"purchase_cost"`
	WarrantyUntil *string   `json:"warranty_until"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	// Tickets sin resolver vinculados a este equipo.
	OpenTickets int `json:"open_tickets"`
}

const assetSelect = `
	SELECT a.id, a.tag, a.name, a.category, a.model, a.serial, a.state, u.id, u.name, a.location,
	       to_char(a.purchase_date, 'YYYY-MM-DD'), a.purchase_cost::float8, to_char(a.warranty_until, 'YYYY-MM-DD'),
	       a.notes, a.created_at, a.updated_at,
	       (SELECT count(*) FROM ticket_assets ta JOIN tickets t ON t.id = ta.ticket_id
	        WHERE ta.asset_id = a.id AND t.status NOT IN ('resolved', 'closed'))
	FROM assets a
	LEFT JOIN users u ON u.id = a.assigned_to`

func scanAsset(row pgx.Row) (Asset, error) {
	var a Asset
	var uid *int64
	var uname *string
	err := row.Scan(&a.ID, &a.Tag, &a.Name, &a.Category, &a.Model, &a.Serial, &a.State, &uid, &uname, &a.Location,
		&a.PurchaseDate, &a.PurchaseCost, &a.WarrantyUntil, &a.Notes, &a.CreatedAt, &a.UpdatedAt, &a.OpenTickets)
	if uid != nil {
		a.AssignedTo = &UserRef{ID: *uid, Name: *uname}
	}
	return a, err
}

func addAssetEvent(ctx context.Context, q querier, assetID, actorID int64, kind, oldValue, newValue string) error {
	_, err := q.Exec(ctx, `
		INSERT INTO asset_events (asset_id, actor_id, kind, old_value, new_value)
		VALUES ($1, NULLIF($2, 0), $3, $4, $5)`, assetID, actorID, kind, oldValue, newValue)
	return err
}

// assetUsable indica si el usuario puede vincular el equipo a un ticket de `requester`:
// el equipo debe ser de su empresa y, si quien lo vincula no es del equipo de soporte,
// debe estar asignado a esa persona.
func (s *Server) assetUsable(ctx context.Context, claims auth.Claims, requester, assetID int64) (bool, error) {
	var assigned *int64
	err := s.db.QueryRow(ctx, `SELECT assigned_to FROM assets WHERE id = $1 AND org_id = $2`, assetID, claims.OrgID).Scan(&assigned)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if claims.IsStaff() {
		return true, nil
	}
	return assigned != nil && *assigned == requester, nil
}

// listAssets admite ?q=&state=&category=&assigned=me|none|<id>&warranty=expiring|expired&before=&limit=
func (s *Server) listAssets(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	q := r.URL.Query()
	var where []string
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	where = append(where, "a.org_id = "+arg(claims.OrgID))
	if v := q.Get("state"); v != "" {
		if !slices.Contains(assetStates, v) {
			writeError(w, http.StatusBadRequest, "state inválido")
			return
		}
		where = append(where, "a.state = "+arg(v))
	}
	if v := q.Get("category"); v != "" {
		if !slices.Contains(assetCategories, v) {
			writeError(w, http.StatusBadRequest, "category inválida")
			return
		}
		where = append(where, "a.category = "+arg(v))
	}
	switch v := q.Get("assigned"); v {
	case "":
	case "me":
		where = append(where, "a.assigned_to = "+arg(claims.UserID()))
	case "none":
		where = append(where, "a.assigned_to IS NULL")
	default:
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "assigned inválido")
			return
		}
		where = append(where, "a.assigned_to = "+arg(id))
	}
	switch q.Get("warranty") {
	case "":
	case "expiring":
		where = append(where, fmt.Sprintf(`a.state <> 'retired' AND a.warranty_until BETWEEN current_date AND current_date + %d`, warrantyWarnDays))
	case "expired":
		where = append(where, `a.state <> 'retired' AND a.warranty_until < current_date`)
	default:
		writeError(w, http.StatusBadRequest, "warranty inválido")
		return
	}
	if v := strings.TrimSpace(q.Get("q")); v != "" {
		p := arg("%" + escapeLike(v) + "%")
		where = append(where, fmt.Sprintf(`(a.tag ILIKE %[1]s OR a.name ILIKE %[1]s OR a.model ILIKE %[1]s OR a.serial ILIKE %[1]s OR u.name ILIKE %[1]s)`, p))
	}
	if v := q.Get("before"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "before inválido")
			return
		}
		where = append(where, "a.id < "+arg(id))
	}
	limit := defaultPageSize
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxPageSize {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("limit debe estar entre 1 y %d", maxPageSize))
			return
		}
		limit = n
	}
	sql := assetSelect + " WHERE " + strings.Join(where, " AND ") + " ORDER BY a.id DESC LIMIT " + arg(limit+1)
	rows, err := s.db.Query(r.Context(), sql, args...)
	if err != nil {
		internalError(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Asset, error) { return scanAsset(row) })
	if err != nil {
		internalError(w, err)
		return
	}
	var next *int64
	if len(items) > limit {
		items = items[:limit]
		next = &items[limit-1].ID
	}
	if items == nil {
		items = []Asset{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "next_cursor": next})
}

type assetInput struct {
	Tag           string   `json:"tag"`
	Name          string   `json:"name"`
	Category      string   `json:"category"`
	Model         string   `json:"model"`
	Serial        string   `json:"serial"`
	State         string   `json:"state"`
	AssignedTo    *int64   `json:"assigned_to"`
	Location      string   `json:"location"`
	PurchaseDate  string   `json:"purchase_date"`
	PurchaseCost  *float64 `json:"purchase_cost"`
	WarrantyUntil string   `json:"warranty_until"`
	Notes         string   `json:"notes"`
}

func (s *Server) readAsset(w http.ResponseWriter, r *http.Request) (assetInput, bool) {
	claims := claimsFrom(r)
	var in assetInput
	if !decodeJSON(w, r, &in) {
		return in, false
	}
	in.Tag = strings.TrimSpace(in.Tag)
	in.Name = strings.TrimSpace(in.Name)
	in.Model = strings.TrimSpace(in.Model)
	in.Serial = strings.TrimSpace(in.Serial)
	in.Location = strings.TrimSpace(in.Location)
	if in.Category == "" {
		in.Category = "computer"
	}
	if in.State == "" {
		in.State = "in_stock"
	}
	validDate := func(v string) bool {
		if v == "" {
			return true
		}
		_, err := time.Parse("2006-01-02", v)
		return err == nil
	}
	errs := validationErrors{}
	errs.check(in.Tag != "", "tag", "es obligatorio")
	errs.check(len(in.Tag) <= 40, "tag", "debe tener como máximo 40 caracteres")
	errs.check(in.Name != "", "name", "es obligatorio")
	errs.check(len(in.Name) <= 120, "name", "debe tener como máximo 120 caracteres")
	errs.check(slices.Contains(assetCategories, in.Category), "category", "valor inválido")
	errs.check(slices.Contains(assetStates, in.State), "state", "valor inválido")
	errs.check(len(in.Model) <= 120, "model", "debe tener como máximo 120 caracteres")
	errs.check(len(in.Serial) <= 120, "serial", "debe tener como máximo 120 caracteres")
	errs.check(len(in.Location) <= 120, "location", "debe tener como máximo 120 caracteres")
	errs.check(len(in.Notes) <= 5000, "notes", "es demasiado largo")
	errs.check(validDate(in.PurchaseDate), "purchase_date", "fecha inválida (AAAA-MM-DD)")
	errs.check(validDate(in.WarrantyUntil), "warranty_until", "fecha inválida (AAAA-MM-DD)")
	errs.check(in.PurchaseCost == nil || (*in.PurchaseCost >= 0 && *in.PurchaseCost < 1e9), "purchase_cost", "valor inválido")
	errs.check(!(in.State == "retired" && in.AssignedTo != nil), "assigned_to", "un equipo dado de baja no puede estar asignado")
	if errs.write(w) {
		return in, false
	}
	if in.AssignedTo != nil {
		if exists, err := s.userExists(r.Context(), claims.OrgID, *in.AssignedTo, false); err != nil {
			internalError(w, err)
			return in, false
		} else if !exists {
			validationErrors{"assigned_to": "el usuario no existe"}.write(w)
			return in, false
		}
	}
	// Asignarlo lo deja en uso y quitar al responsable lo regresa al almacén, salvo que se indique otro estado.
	if in.AssignedTo != nil && in.State == "in_stock" {
		in.State = "in_use"
	}
	if in.AssignedTo == nil && in.State == "in_use" {
		in.State = "in_stock"
	}
	return in, true
}

func (s *Server) createAsset(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	in, ok := s.readAsset(w, r)
	if !ok {
		return
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var id int64
	err = tx.QueryRow(r.Context(), `
		INSERT INTO assets (org_id, tag, name, category, model, serial, state, assigned_to, location,
		                    purchase_date, purchase_cost, warranty_until, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10, '')::date, $11, NULLIF($12, '')::date, $13)
		RETURNING id`,
		claims.OrgID, in.Tag, in.Name, in.Category, in.Model, in.Serial, in.State, in.AssignedTo, in.Location,
		in.PurchaseDate, in.PurchaseCost, in.WarrantyUntil, in.Notes).Scan(&id)
	if isUnique(err) {
		validationErrors{"tag": "ya existe un equipo con esa etiqueta"}.write(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	if err := addAssetEvent(r.Context(), tx, id, claims.UserID(), "created", "", ""); err != nil {
		internalError(w, err)
		return
	}
	a, err := scanAsset(tx.QueryRow(r.Context(), assetSelect+" WHERE a.id = $1", id))
	if err != nil {
		internalError(w, err)
		return
	}
	if a.AssignedTo != nil {
		if err := addAssetEvent(r.Context(), tx, id, claims.UserID(), "assigned", "", a.AssignedTo.Name); err != nil {
			internalError(w, err)
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

type AssetEvent struct {
	ID        int64     `json:"id"`
	Actor     *UserRef  `json:"actor"`
	Kind      string    `json:"kind"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	CreatedAt time.Time `json:"created_at"`
}

type AssetTicket struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	Priority  string    `json:"priority"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Server) loadAsset(w http.ResponseWriter, r *http.Request, id int64) (Asset, bool) {
	a, err := scanAsset(s.db.QueryRow(r.Context(), assetSelect+" WHERE a.id = $1 AND a.org_id = $2", id, claimsFrom(r).OrgID))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "equipo no encontrado")
		return a, false
	}
	if err != nil {
		internalError(w, err)
		return a, false
	}
	return a, true
}

// getAsset devuelve el equipo con su actividad y los tickets que lo mencionan.
func (s *Server) getAsset(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	a, ok := s.loadAsset(w, r, id)
	if !ok {
		return
	}
	rows, err := s.db.Query(r.Context(), `
		SELECT e.id, u.id, u.name, e.kind, e.old_value, e.new_value, e.created_at
		FROM asset_events e LEFT JOIN users u ON u.id = e.actor_id
		WHERE e.asset_id = $1 ORDER BY e.id DESC`, id)
	if err != nil {
		internalError(w, err)
		return
	}
	events, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (AssetEvent, error) {
		var e AssetEvent
		var uid *int64
		var uname *string
		err := row.Scan(&e.ID, &uid, &uname, &e.Kind, &e.OldValue, &e.NewValue, &e.CreatedAt)
		if uid != nil {
			e.Actor = &UserRef{ID: *uid, Name: *uname}
		}
		return e, err
	})
	if err != nil {
		internalError(w, err)
		return
	}
	rows, err = s.db.Query(r.Context(), `
		SELECT t.id, t.title, t.status, t.priority, t.kind, t.created_at
		FROM ticket_assets ta JOIN tickets t ON t.id = ta.ticket_id
		WHERE ta.asset_id = $1 ORDER BY t.id DESC LIMIT 100`, id)
	if err != nil {
		internalError(w, err)
		return
	}
	tickets, err := pgx.CollectRows(rows, pgx.RowToStructByPos[AssetTicket])
	if err != nil {
		internalError(w, err)
		return
	}
	if events == nil {
		events = []AssetEvent{}
	}
	if tickets == nil {
		tickets = []AssetTicket{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"asset": a, "events": events, "tickets": tickets})
}

func (s *Server) updateAsset(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	in, ok := s.readAsset(w, r)
	if !ok {
		return
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	before, err := scanAsset(tx.QueryRow(r.Context(),
		assetSelect+" WHERE a.id = $1 AND a.org_id = $2 FOR UPDATE OF a", id, claims.OrgID))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "equipo no encontrado")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	_, err = tx.Exec(r.Context(), `
		UPDATE assets SET tag = $3, name = $4, category = $5, model = $6, serial = $7, state = $8, assigned_to = $9,
			location = $10, purchase_date = NULLIF($11, '')::date, purchase_cost = $12,
			warranty_until = NULLIF($13, '')::date, notes = $14, updated_at = now()
		WHERE id = $1 AND org_id = $2`,
		id, claims.OrgID, in.Tag, in.Name, in.Category, in.Model, in.Serial, in.State, in.AssignedTo, in.Location,
		in.PurchaseDate, in.PurchaseCost, in.WarrantyUntil, in.Notes)
	if isUnique(err) {
		validationErrors{"tag": "ya existe un equipo con esa etiqueta"}.write(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	after, err := scanAsset(tx.QueryRow(r.Context(), assetSelect+" WHERE a.id = $1", id))
	if err != nil {
		internalError(w, err)
		return
	}

	actor := claims.UserID()
	type change struct{ kind, old, new string }
	var changes []change
	if before.State != after.State {
		changes = append(changes, change{"state", before.State, after.State})
	}
	name := func(u *UserRef) string {
		if u == nil {
			return ""
		}
		return u.Name
	}
	if name(before.AssignedTo) != name(after.AssignedTo) || (before.AssignedTo == nil) != (after.AssignedTo == nil) {
		changes = append(changes, change{"assigned", name(before.AssignedTo), name(after.AssignedTo)})
	}
	if before.Location != after.Location {
		changes = append(changes, change{"location", before.Location, after.Location})
	}
	var others []string
	str := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	for _, f := range []struct {
		key      string
		old, new string
	}{
		{"tag", before.Tag, after.Tag}, {"name", before.Name, after.Name}, {"category", before.Category, after.Category},
		{"model", before.Model, after.Model}, {"serial", before.Serial, after.Serial},
		{"purchase_date", str(before.PurchaseDate), str(after.PurchaseDate)},
		{"warranty_until", str(before.WarrantyUntil), str(after.WarrantyUntil)},
		{"notes", before.Notes, after.Notes},
	} {
		if f.old != f.new {
			others = append(others, f.key)
		}
	}
	if (before.PurchaseCost == nil) != (after.PurchaseCost == nil) ||
		(before.PurchaseCost != nil && *before.PurchaseCost != *after.PurchaseCost) {
		others = append(others, "purchase_cost")
	}
	if len(others) > 0 {
		changes = append(changes, change{"updated", "", strings.Join(others, ",")})
	}
	for _, c := range changes {
		if err := addAssetEvent(r.Context(), tx, id, actor, c.kind, c.old, c.new); err != nil {
			internalError(w, err)
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, after)
}

func (s *Server) deleteAsset(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	tag, err := s.db.Exec(r.Context(), `DELETE FROM assets WHERE id = $1 AND org_id = $2`, id, claimsFrom(r).OrgID)
	if err != nil {
		internalError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "equipo no encontrado")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// myAssets son los equipos asignados a la persona que consulta (para reportar un problema con uno).
func (s *Server) myAssets(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	rows, err := s.db.Query(r.Context(), assetSelect+
		" WHERE a.org_id = $1 AND a.assigned_to = $2 AND a.state <> 'retired' ORDER BY lower(a.name)",
		claims.OrgID, claims.UserID())
	if err != nil {
		internalError(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Asset, error) { return scanAsset(row) })
	if err != nil {
		internalError(w, err)
		return
	}
	if items == nil {
		items = []Asset{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type assetSummary struct {
	Total      int            `json:"total"`
	TotalValue float64        `json:"total_value"`
	ByState    map[string]int `json:"by_state"`
	ByCategory map[string]int `json:"by_category"`
	// Garantías que vencen en los próximos días y las que ya vencieron (sin contar equipos dados de baja).
	WarrantyExpiring int `json:"warranty_expiring"`
	WarrantyExpired  int `json:"warranty_expired"`
	// Equipos con algún ticket sin resolver.
	WithOpenTickets int `json:"with_open_tickets"`
	Unassigned      int `json:"unassigned_in_use"`
}

// assetsSummary calcula los números del panel de activos.
func (s *Server) assetsSummary(w http.ResponseWriter, r *http.Request) {
	orgID := claimsFrom(r).OrgID
	sum := assetSummary{ByState: map[string]int{}, ByCategory: map[string]int{}}
	for _, st := range assetStates {
		sum.ByState[st] = 0
	}
	for _, c := range assetCategories {
		sum.ByCategory[c] = 0
	}
	rows, err := s.db.Query(r.Context(), `
		SELECT 'state', state, count(*), coalesce(sum(purchase_cost) FILTER (WHERE state <> 'retired'), 0)::float8
		FROM assets WHERE org_id = $1 GROUP BY state
		UNION ALL
		SELECT 'category', category, count(*), 0 FROM assets WHERE org_id = $1 AND state <> 'retired' GROUP BY category`, orgID)
	if err != nil {
		internalError(w, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var kind, key string
		var n int
		var value float64
		if err := rows.Scan(&kind, &key, &n, &value); err != nil {
			internalError(w, err)
			return
		}
		if kind == "state" {
			sum.ByState[key] = n
			sum.Total += n
			sum.TotalValue += value
		} else {
			sum.ByCategory[key] = n
		}
	}
	if err := rows.Err(); err != nil {
		internalError(w, err)
		return
	}
	if err := s.db.QueryRow(r.Context(), fmt.Sprintf(`
		SELECT
			count(*) FILTER (WHERE state <> 'retired' AND warranty_until BETWEEN current_date AND current_date + %d),
			count(*) FILTER (WHERE state <> 'retired' AND warranty_until < current_date),
			count(*) FILTER (WHERE EXISTS (SELECT 1 FROM ticket_assets ta JOIN tickets t ON t.id = ta.ticket_id
			                               WHERE ta.asset_id = assets.id AND t.status NOT IN ('resolved', 'closed'))),
			count(*) FILTER (WHERE state = 'in_use' AND assigned_to IS NULL)
		FROM assets WHERE org_id = $1`, warrantyWarnDays), orgID).Scan(
		&sum.WarrantyExpiring, &sum.WarrantyExpired, &sum.WithOpenTickets, &sum.Unassigned); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

// --- Vínculo entre tickets y equipos ---

func (s *Server) listTicketAssets(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, ok := s.loadTicket(w, r, id); !ok {
		return
	}
	rows, err := s.db.Query(r.Context(), assetSelect+
		" JOIN ticket_assets ta ON ta.asset_id = a.id WHERE ta.ticket_id = $1 AND a.org_id = $2 ORDER BY a.id", id, claimsFrom(r).OrgID)
	if err != nil {
		internalError(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Asset, error) { return scanAsset(row) })
	if err != nil {
		internalError(w, err)
		return
	}
	if items == nil {
		items = []Asset{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) linkTicketAsset(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		AssetID int64 `json:"asset_id"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	t, ok := s.loadTicket(w, r, id)
	if !ok {
		return
	}
	if exists, err := s.assetUsable(r.Context(), claims, t.Requester.ID, in.AssetID); err != nil {
		internalError(w, err)
		return
	} else if !exists {
		validationErrors{"asset_id": "el equipo no existe"}.write(w)
		return
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	tag, err := tx.Exec(r.Context(),
		`INSERT INTO ticket_assets (ticket_id, asset_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, id, in.AssetID)
	if err != nil {
		internalError(w, err)
		return
	}
	if tag.RowsAffected() > 0 {
		if err := addAssetEvent(r.Context(), tx, in.AssetID, claims.UserID(), "ticket", "", "#"+strconv.FormatInt(id, 10)); err != nil {
			internalError(w, err)
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	a, err := scanAsset(s.db.QueryRow(r.Context(), assetSelect+" WHERE a.id = $1", in.AssetID))
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) unlinkTicketAsset(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	assetID, err := strconv.ParseInt(r.PathValue("assetId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if _, ok := s.loadTicket(w, r, id); !ok {
		return
	}
	tag, err := s.db.Exec(r.Context(), `
		DELETE FROM ticket_assets ta USING assets a
		WHERE ta.ticket_id = $1 AND ta.asset_id = $2 AND a.id = ta.asset_id AND a.org_id = $3`,
		id, assetID, claimsFrom(r).OrgID)
	if err != nil {
		internalError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "ese equipo no está vinculado al ticket")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
