package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Article struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Category  string    `json:"category"`
	Published bool      `json:"published"`
	Author    *UserRef  `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Empresa dueña del artículo: el centro de ayuda de cada empresa es público.
	Org   orgChoice `json:"org"`
	orgID int64
}

const articleSelect = `
	SELECT a.id, a.title, a.body, a.category, a.published, u.id, u.name, a.created_at, a.updated_at,
	       o.slug, o.name, a.org_id
	FROM articles a
	JOIN organizations o ON o.id = a.org_id
	LEFT JOIN users u ON u.id = a.author_id`

func scanArticle(row pgx.Row) (Article, error) {
	var a Article
	var authorID *int64
	var authorName *string
	err := row.Scan(&a.ID, &a.Title, &a.Body, &a.Category, &a.Published, &authorID, &authorName, &a.CreatedAt, &a.UpdatedAt,
		&a.Org.Slug, &a.Org.Name, &a.orgID)
	if authorID != nil {
		a.Author = &UserRef{ID: *authorID, Name: *authorName}
	}
	return a, err
}

// listArticles busca en la base de conocimiento (?q=texto) de una empresa: la indicada con ?org=<slug>
// o, si no, la del usuario (o la de la plataforma para visitantes). Los borradores solo los ve su equipo.
func (s *Server) listArticles(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	orgID := claims.OrgID
	if slug := r.URL.Query().Get("org"); slug != "" || orgID == 0 {
		org, ok := s.portalOrg(w, r, slug)
		if !ok {
			return
		}
		orgID = org.ID
	}
	staff := claims.IsStaff() && claims.OrgID == orgID
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	order := "a.updated_at DESC"
	if q != "" {
		order = "ts_rank(a.search, websearch_to_tsquery('spanish', $2)) DESC, a.updated_at DESC"
	}
	rows, err := s.db.Query(r.Context(), articleSelect+`
		WHERE a.org_id = $3 AND ($1 OR a.published) AND NOT o.suspended
		  AND ($2 = '' OR a.search @@ websearch_to_tsquery('spanish', $2))
		ORDER BY `+order+`
		LIMIT 100`, staff, q, orgID)
	if err != nil {
		internalError(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Article, error) { return scanArticle(row) })
	if err != nil {
		internalError(w, err)
		return
	}
	if items == nil {
		items = []Article{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getArticle(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	a, err := scanArticle(s.db.QueryRow(r.Context(), articleSelect+" WHERE a.id = $1 AND NOT o.suspended", id))
	claims := claimsFrom(r)
	ownTeam := claims.IsStaff() && claims.OrgID == a.orgID
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !a.Published && !ownTeam) {
		writeError(w, http.StatusNotFound, "artículo no encontrado")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

type articleInput struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	Category  string `json:"category"`
	Published bool   `json:"published"`
}

func readArticle(w http.ResponseWriter, r *http.Request) (articleInput, bool) {
	var in articleInput
	if !decodeJSON(w, r, &in) {
		return in, false
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Category = strings.TrimSpace(in.Category)
	errs := validationErrors{}
	errs.check(in.Title != "", "title", "es obligatorio")
	errs.check(len(in.Title) <= 200, "title", "debe tener como máximo 200 caracteres")
	errs.check(len(in.Body) <= 100000, "body", "es demasiado largo")
	errs.check(len(in.Category) <= 100, "category", "debe tener como máximo 100 caracteres")
	return in, !errs.write(w)
}

func (s *Server) createArticle(w http.ResponseWriter, r *http.Request) {
	in, ok := readArticle(w, r)
	if !ok {
		return
	}
	var id int64
	if err := s.db.QueryRow(r.Context(), `
		INSERT INTO articles (title, body, category, published, author_id, org_id)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		in.Title, in.Body, in.Category, in.Published, claimsFrom(r).UserID(), claimsFrom(r).OrgID).Scan(&id); err != nil {
		internalError(w, err)
		return
	}
	a, err := scanArticle(s.db.QueryRow(r.Context(), articleSelect+" WHERE a.id = $1", id))
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (s *Server) updateArticle(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	in, ok := readArticle(w, r)
	if !ok {
		return
	}
	tag, err := s.db.Exec(r.Context(), `
		UPDATE articles SET title = $2, body = $3, category = $4, published = $5, updated_at = now()
		WHERE id = $1 AND org_id = $6`, id, in.Title, in.Body, in.Category, in.Published, claimsFrom(r).OrgID)
	if err != nil {
		internalError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "artículo no encontrado")
		return
	}
	a, err := scanArticle(s.db.QueryRow(r.Context(), articleSelect+" WHERE a.id = $1", id))
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) deleteArticle(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	tag, err := s.db.Exec(r.Context(), `DELETE FROM articles WHERE id = $1 AND org_id = $2`, id, claimsFrom(r).OrgID)
	if err != nil {
		internalError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "artículo no encontrado")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
