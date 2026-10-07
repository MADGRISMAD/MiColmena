package api

import (
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/MADGRISMAD/MiColmena/backend/internal/auth"
)

type User struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Active bool   `json:"active"`
	// EmailNotifications: además de la campana, recibir avisos por correo.
	EmailNotifications bool      `json:"email_notifications"`
	CreatedAt          time.Time `json:"created_at"`
	// Permanent: administrador que no se puede desactivar ni bajar de rol (ver permanent.go).
	Permanent bool  `json:"permanent"`
	OrgID     int64 `json:"org_id"`
}

const userColumns = `id, name, email, role, active, email_notifications, created_at, org_id`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.Active, &u.EmailNotifications, &u.CreatedAt, &u.OrgID)
	u.Permanent = isPermanentAdmin(u)
	return u, err
}

type authResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

// register crea una cuenta de cliente en el portal de una empresa (?org=<slug> en el cuerpo).
// Sin empresa, la cuenta es del soporte de la propia plataforma.
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Org      string `json:"org"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.TrimSpace(in.Email)

	errs := validationErrors{}
	checkName(errs, in.Name)
	checkEmail(errs, in.Email)
	checkPassword(errs, "password", in.Password)
	if errs.write(w) {
		return
	}

	org, ok := s.portalOrg(w, r, in.Org)
	if !ok {
		return
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		internalError(w, err)
		return
	}

	// El registro público siempre crea clientes; los agentes los asigna un administrador.
	u, err := scanUser(s.db.QueryRow(r.Context(), `
		INSERT INTO users (name, email, password_hash, role, org_id)
		VALUES ($1, $2, $3, 'customer', $4)
		RETURNING `+userColumns,
		in.Name, in.Email, hash, org.ID))
	if isUniqueViolation(err) {
		writeError(w, http.StatusConflict, "ese email ya está registrado")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	s.respondWithToken(w, http.StatusCreated, u)
}

// orgChoice es una empresa entre las que el email tiene cuenta, para elegir al iniciar sesión.
type orgChoice struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// login inicia sesión. Si el email y la contraseña coinciden en varias empresas y no se indica
// cuál (campo org), responde 409 con la lista para que la persona elija.
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Org      string `json:"org"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}

	// El bloqueo es por el email que se escribe, exista o no la cuenta, para no revelar cuáles existen.
	// Mientras dura, ni siquiera la contraseña correcta entra: así el bloqueo frena de verdad la fuerza bruta.
	key := strings.ToLower(strings.TrimSpace(in.Email))
	if locked, retry := s.loginLock.Locked(key); locked {
		tooManyRequests(w, retry)
		return
	}

	rows, err := s.db.Query(r.Context(), `
		SELECT `+prefixed("u.", userColumns)+`, u.password_hash, o.slug, o.name, o.suspended
		FROM users u JOIN organizations o ON o.id = u.org_id
		WHERE lower(u.email) = lower($1) AND ($2 = '' OR o.slug = $2)
		ORDER BY o.id`,
		strings.TrimSpace(in.Email), strings.ToLower(strings.TrimSpace(in.Org)))
	if err != nil {
		internalError(w, err)
		return
	}
	type candidate struct {
		user      User
		org       orgChoice
		suspended bool
	}
	var matches []candidate
	for rows.Next() {
		var c candidate
		var hash string
		u := &c.user
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.Active, &u.EmailNotifications, &u.CreatedAt, &u.OrgID,
			&hash, &c.org.Slug, &c.org.Name, &c.suspended); err != nil {
			rows.Close()
			internalError(w, err)
			return
		}
		// Se comprueba la contraseña de cada cuenta: así solo se revelan las empresas de quien la sabe.
		if auth.CheckPassword(hash, in.Password) {
			u.Permanent = isPermanentAdmin(*u)
			matches = append(matches, c)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		internalError(w, err)
		return
	}
	if len(matches) == 0 {
		// Mismo costo de bcrypt aunque el email no exista, para no revelar qué cuentas hay.
		auth.CheckPassword(dummyHash, in.Password)
		s.loginLock.Fail(key)
		writeError(w, http.StatusUnauthorized, "email o contraseña incorrectos")
		return
	}
	s.loginLock.Reset(key)
	if len(matches) > 1 {
		choices := make([]orgChoice, len(matches))
		for i, m := range matches {
			choices[i] = m.org
		}
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":         "tienes cuenta en varias empresas: elige a cuál entrar",
			"organizations": choices,
		})
		return
	}

	m := matches[0]
	if m.suspended {
		writeError(w, http.StatusForbidden, "la cuenta de esta empresa está suspendida; escríbenos para reactivarla")
		return
	}
	if !m.user.Active {
		writeError(w, http.StatusForbidden, "esta cuenta está desactivada; habla con un administrador")
		return
	}
	// Una sesión por agente: entrar cierra las sesiones de otros dispositivos.
	// Así una licencia de agente no se comparte entre varias personas.
	if m.user.Role == auth.RoleAgent || m.user.Role == auth.RoleAdmin {
		if _, err := s.db.Exec(r.Context(),
			`UPDATE users SET tokens_valid_after = date_trunc('second', now()) WHERE id = $1`, m.user.ID); err != nil {
			internalError(w, err)
			return
		}
	}
	s.respondWithToken(w, http.StatusOK, m.user)
}

// dummyHash iguala el tiempo de respuesta cuando el email no existe.
var dummyHash, _ = auth.HashPassword("micolmena-no-existe")

// prefixed antepone un alias a cada columna de una lista: "id, name" → "u.id, u.name".
func prefixed(alias, columns string) string {
	parts := strings.Split(columns, ",")
	for i, p := range parts {
		parts[i] = alias + strings.TrimSpace(p)
	}
	return strings.Join(parts, ", ")
}

func (s *Server) respondWithToken(w http.ResponseWriter, status int, u User) {
	token, expires, err := s.tokens.Issue(u.ID, u.Role)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, status, authResponse{Token: token, ExpiresAt: expires, User: u})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u, err := scanUser(s.db.QueryRow(r.Context(),
		`SELECT `+userColumns+` FROM users WHERE id = $1`, claimsFrom(r).UserID()))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusUnauthorized, "el usuario ya no existe")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// listUsers devuelve usuarios, opcionalmente filtrados por rol (?role=agent).
func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	if role != "" && !validRole(role) {
		writeError(w, http.StatusBadRequest, "rol inválido")
		return
	}

	search := strings.TrimSpace(r.URL.Query().Get("q"))
	// Por defecto solo los activos; ?active=all incluye los desactivados.
	all := r.URL.Query().Get("active") == "all"
	rows, err := s.db.Query(r.Context(), `
		SELECT `+userColumns+` FROM users
		WHERE org_id = $4
		  AND ($1 = '' OR role = $1)
		  AND ($2 OR active)
		  AND ($3 = '' OR name ILIKE '%' || $3 || '%' OR email ILIKE '%' || $3 || '%')
		ORDER BY active DESC, name
		LIMIT 500`, role, all, escapeLike(search), claimsFrom(r).OrgID)
	if err != nil {
		internalError(w, err)
		return
	}
	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (User, error) { return scanUser(row) })
	if err != nil {
		internalError(w, err)
		return
	}
	if users == nil {
		users = []User{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": users})
}

func (s *Server) updateUserRole(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Role string `json:"role"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validRole(in.Role) {
		writeError(w, http.StatusUnprocessableEntity, "rol inválido")
		return
	}
	if id == claimsFrom(r).UserID() && in.Role != auth.RoleAdmin {
		writeError(w, http.StatusUnprocessableEntity, "no puedes quitarte el rol de administrador")
		return
	}
	if msg, err := s.permanentChange(r.Context(), claimsFrom(r).OrgID, claimsFrom(r).UserID(), id, &in.Role, nil, nil, nil); err != nil {
		internalError(w, err)
		return
	} else if msg != "" {
		writeError(w, http.StatusForbidden, msg)
		return
	}

	orgID := claimsFrom(r).OrgID
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	if isStaffRole(in.Role) {
		if msg, err := agentLimit(r.Context(), tx, orgID, id); err != nil {
			internalError(w, err)
			return
		} else if msg != "" {
			validationErrors{"role": msg}.write(w)
			return
		}
	}
	u, err := scanUser(tx.QueryRow(r.Context(),
		`UPDATE users SET role = $2 WHERE id = $1 AND org_id = $3 RETURNING `+userColumns, id, in.Role, orgID))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "usuario no encontrado")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func isStaffRole(role string) bool { return role == auth.RoleAgent || role == auth.RoleAdmin }

func validRole(role string) bool {
	return role == auth.RoleAdmin || role == auth.RoleAgent || role == auth.RoleCustomer
}

func checkName(errs validationErrors, name string) {
	errs.check(notBlank(name), "name", "es obligatorio")
	errs.check(len(name) <= 100, "name", "debe tener como máximo 100 caracteres")
}

func checkEmail(errs validationErrors, email string) {
	addr, err := mail.ParseAddress(email)
	errs.check(err == nil && addr.Address == email, "email", "no es un email válido")
}

func checkPassword(errs validationErrors, field, password string) {
	errs.check(len(password) >= 8, field, "debe tener al menos 8 caracteres")
	errs.check(len(password) <= 72, field, "debe tener como máximo 72 caracteres")
}

// escapeLike escapa los comodines de LIKE para buscar el texto tal cual.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// createUser lo usa un administrador para dar de alta agentes (o clientes) con una contraseña inicial.
func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Role     string `json:"role"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.TrimSpace(in.Email)
	errs := validationErrors{}
	checkName(errs, in.Name)
	checkEmail(errs, in.Email)
	checkPassword(errs, "password", in.Password)
	errs.check(validRole(in.Role), "role", "rol inválido")
	if errs.write(w) {
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		internalError(w, err)
		return
	}
	orgID := claimsFrom(r).OrgID
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	if isStaffRole(in.Role) {
		if msg, err := agentLimit(r.Context(), tx, orgID, 0); err != nil {
			internalError(w, err)
			return
		} else if msg != "" {
			validationErrors{"role": msg}.write(w)
			return
		}
	}
	u, err := scanUser(tx.QueryRow(r.Context(), `
		INSERT INTO users (name, email, password_hash, role, org_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+userColumns, in.Name, in.Email, hash, in.Role, orgID))
	if isUniqueViolation(err) {
		validationErrors{"email": "ese email ya está registrado"}.write(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

// updateUser permite a un administrador cambiar nombre, email, rol, contraseña o desactivar la cuenta.
// Al desactivar a un agente, sus tickets pendientes quedan sin asignar para que no se pierdan.
func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Name     *string `json:"name"`
		Email    *string `json:"email"`
		Role     *string `json:"role"`
		Active   *bool   `json:"active"`
		Password *string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	self := id == claimsFrom(r).UserID()

	errs := validationErrors{}
	var sets []string
	var args []any
	set := func(expr string, v any) {
		args = append(args, v)
		sets = append(sets, strings.ReplaceAll(expr, "?", "$"+strconv.Itoa(len(args))))
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		checkName(errs, name)
		set("name = ?", name)
	}
	if in.Email != nil {
		email := strings.TrimSpace(*in.Email)
		checkEmail(errs, email)
		set("email = ?", email)
	}
	if in.Role != nil {
		errs.check(validRole(*in.Role), "role", "rol inválido")
		errs.check(!self || *in.Role == auth.RoleAdmin, "role", "no puedes quitarte el rol de administrador")
		set("role = ?", *in.Role)
	}
	if in.Active != nil {
		errs.check(!self || *in.Active, "active", "no puedes desactivar tu propia cuenta")
		set("active = ?", *in.Active)
	}
	if in.Password != nil {
		checkPassword(errs, "password", *in.Password)
		if len(errs) == 0 {
			hash, err := auth.HashPassword(*in.Password)
			if err != nil {
				internalError(w, err)
				return
			}
			set("password_hash = ?", hash)
			// Las sesiones abiertas con la contraseña anterior dejan de valer.
			sets = append(sets, "tokens_valid_after = date_trunc('second', now())")
		}
	}
	if errs.write(w) {
		return
	}
	if msg, err := s.permanentChange(r.Context(), claimsFrom(r).OrgID, claimsFrom(r).UserID(), id, in.Role, in.Active, in.Email, in.Password); err != nil {
		internalError(w, err)
		return
	} else if msg != "" {
		writeError(w, http.StatusForbidden, msg)
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback(r.Context())

	orgID := claimsFrom(r).OrgID
	// Si el cambio deja a la persona como agente activo, debe caber en el plan.
	var before User
	before, err = scanUser(tx.QueryRow(r.Context(),
		`SELECT `+userColumns+` FROM users WHERE id = $1 AND org_id = $2`, id, orgID))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "usuario no encontrado")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	role, active := before.Role, before.Active
	if in.Role != nil {
		role = *in.Role
	}
	if in.Active != nil {
		active = *in.Active
	}
	if isStaffRole(role) && active && !(isStaffRole(before.Role) && before.Active) {
		if msg, err := agentLimit(r.Context(), tx, orgID, id); err != nil {
			internalError(w, err)
			return
		} else if msg != "" {
			field := "role"
			if in.Role == nil {
				field = "active"
			}
			validationErrors{field: msg}.write(w)
			return
		}
	}

	var u User
	if len(sets) == 0 {
		u = before
	} else {
		args = append(args, id, orgID)
		u, err = scanUser(tx.QueryRow(r.Context(),
			`UPDATE users SET `+strings.Join(sets, ", ")+` WHERE id = $`+strconv.Itoa(len(args)-1)+
				` AND org_id = $`+strconv.Itoa(len(args))+` RETURNING `+userColumns,
			args...))
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "usuario no encontrado")
		return
	}
	if isUniqueViolation(err) {
		validationErrors{"email": "ese email ya está registrado"}.write(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	if !u.Active || u.Role == auth.RoleCustomer {
		if _, err := tx.Exec(r.Context(), `
			WITH released AS (
				UPDATE tickets SET assignee_id = NULL, updated_at = now()
				WHERE assignee_id = $1 AND status NOT IN ('resolved', 'closed')
				RETURNING id
			)
			INSERT INTO ticket_events (ticket_id, actor_id, kind, old_value, new_value)
			SELECT id, $2, 'assignee', $3, '' FROM released`,
			u.ID, claimsFrom(r).UserID(), u.Name); err != nil {
			internalError(w, err)
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// updateMe cambia el nombre o el email de la propia cuenta. Cambiar el email pide la contraseña actual.
func (s *Server) updateMe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name               *string `json:"name"`
		Email              *string `json:"email"`
		CurrentPassword    string  `json:"current_password"`
		EmailNotifications *bool   `json:"email_notifications"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	id := claimsFrom(r).UserID()
	errs := validationErrors{}
	name, email := "", ""
	if in.Name != nil {
		name = strings.TrimSpace(*in.Name)
		checkName(errs, name)
	}
	if in.Email != nil {
		email = strings.TrimSpace(*in.Email)
		checkEmail(errs, email)
	}
	if errs.write(w) {
		return
	}

	var currentEmail, hash string
	if err := s.db.QueryRow(r.Context(), `SELECT email, password_hash FROM users WHERE id = $1`, id).
		Scan(&currentEmail, &hash); err != nil {
		internalError(w, err)
		return
	}
	if in.Email != nil && email != currentEmail && !auth.CheckPassword(hash, in.CurrentPassword) {
		validationErrors{"current_password": "la contraseña no es correcta"}.write(w)
		return
	}

	u, err := scanUser(s.db.QueryRow(r.Context(), `
		UPDATE users SET
			name  = CASE WHEN $2 = '' THEN name ELSE $2 END,
			email = CASE WHEN $3 = '' THEN email ELSE $3 END,
			email_notifications = coalesce($4, email_notifications)
		WHERE id = $1
		RETURNING `+userColumns, id, name, email, in.EmailNotifications))
	if isUniqueViolation(err) {
		validationErrors{"email": "ese email ya está registrado"}.write(w)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// changePassword cambia la contraseña, cierra las demás sesiones y devuelve un token nuevo.
func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	errs := validationErrors{}
	checkPassword(errs, "new_password", in.NewPassword)
	if errs.write(w) {
		return
	}
	id := claimsFrom(r).UserID()
	var hash string
	if err := s.db.QueryRow(r.Context(), `SELECT password_hash FROM users WHERE id = $1`, id).Scan(&hash); err != nil {
		internalError(w, err)
		return
	}
	if !auth.CheckPassword(hash, in.CurrentPassword) {
		validationErrors{"current_password": "la contraseña no es correcta"}.write(w)
		return
	}
	newHash, err := auth.HashPassword(in.NewPassword)
	if err != nil {
		internalError(w, err)
		return
	}
	u, err := scanUser(s.db.QueryRow(r.Context(), `
		UPDATE users SET password_hash = $2, tokens_valid_after = date_trunc('second', now())
		WHERE id = $1
		RETURNING `+userColumns, id, newHash))
	if err != nil {
		internalError(w, err)
		return
	}
	s.respondWithToken(w, http.StatusOK, u)
}
