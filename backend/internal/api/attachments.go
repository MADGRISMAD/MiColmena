package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
)

type Attachment struct {
	ID          int64     `json:"id"`
	TicketID    int64     `json:"ticket_id"`
	CommentID   *int64    `json:"comment_id"`
	Uploader    UserRef   `json:"uploader"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	Internal    bool      `json:"internal"`
	CreatedAt   time.Time `json:"created_at"`
	storageKey  string
}

// Un adjunto es interno si pertenece a una nota interna.
const attachmentSelect = `
	SELECT a.id, a.ticket_id, a.comment_id, u.id, u.name, a.filename, a.content_type, a.size,
	       coalesce(c.internal, false), a.created_at, a.storage_key
	FROM attachments a
	JOIN users u ON u.id = a.uploader_id
	LEFT JOIN ticket_comments c ON c.id = a.comment_id`

func scanAttachment(row pgx.Row) (Attachment, error) {
	var a Attachment
	err := row.Scan(&a.ID, &a.TicketID, &a.CommentID, &a.Uploader.ID, &a.Uploader.Name, &a.Filename,
		&a.ContentType, &a.Size, &a.Internal, &a.CreatedAt, &a.storageKey)
	return a, err
}

func (s *Server) listAttachments(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, ok := s.loadTicket(w, r, id); !ok {
		return
	}
	rows, err := s.db.Query(r.Context(), attachmentSelect+`
		WHERE a.ticket_id = $1 AND ($2 OR NOT coalesce(c.internal, false))
		ORDER BY a.id`, id, claimsFrom(r).IsStaff())
	if err != nil {
		internalError(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Attachment, error) { return scanAttachment(row) })
	if err != nil {
		internalError(w, err)
		return
	}
	if items == nil {
		items = []Attachment{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// uploadAttachment recibe un archivo (multipart, campo "file") y opcionalmente "comment_id".
// Sin comment_id, el archivo acompaña la descripción del ticket.
func (s *Server) uploadAttachment(w http.ResponseWriter, r *http.Request) {
	ticketID, ok := pathID(w, r)
	if !ok {
		return
	}
	ticket, ok := s.loadTicket(w, r, ticketID)
	if !ok {
		return
	}
	claims := claimsFrom(r)

	// El servidor limita cada petición a pocos segundos; una subida grande en una conexión lenta necesita más.
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(5 * time.Minute))
	// Margen de 64 KB para las cabeceras multipart y el resto de campos.
	r.Body = http.MaxBytesReader(w, r.Body, s.maxUpload+64<<10)
	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "se esperaba un formulario multipart")
		return
	}

	var (
		commentID   *int64
		filename    string
		contentType string
		size        int64
		key         string
	)
	cleanup := func() {
		if key != "" {
			os.Remove(s.storagePath(key))
		}
	}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			cleanup()
			uploadReadError(w, err)
			return
		}
		switch part.FormName() {
		case "comment_id":
			raw, _ := io.ReadAll(io.LimitReader(part, 32))
			id, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
			if err != nil {
				cleanup()
				validationErrors{"comment_id": "id inválido"}.write(w)
				return
			}
			commentID = &id
		case "file":
			if key != "" {
				cleanup()
				validationErrors{"file": "envía un solo archivo por petición"}.write(w)
				return
			}
			filename = cleanFilename(part.FileName())
			key, contentType, size, err = s.store(part)
			if err != nil {
				cleanup()
				uploadReadError(w, err)
				return
			}
		}
		part.Close()
	}
	if key == "" {
		validationErrors{"file": "falta el archivo"}.write(w)
		return
	}
	if size > s.maxUpload {
		cleanup()
		writeError(w, http.StatusRequestEntityTooLarge,
			fmt.Sprintf("el archivo supera el máximo de %d MB", s.maxUpload>>20))
		return
	}
	if size == 0 {
		cleanup()
		validationErrors{"file": "el archivo está vacío"}.write(w)
		return
	}

	// Los permisos: el autor del comentario, o el solicitante o un agente para la descripción.
	if commentID != nil {
		var authorID int64
		err := s.db.QueryRow(r.Context(),
			`SELECT author_id FROM ticket_comments WHERE id = $1 AND ticket_id = $2`, *commentID, ticketID,
		).Scan(&authorID)
		if err != nil || authorID != claims.UserID() {
			cleanup()
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				internalError(w, err)
				return
			}
			validationErrors{"comment_id": "solo puedes adjuntar archivos a tus comentarios de este ticket"}.write(w)
			return
		}
	} else if !claims.IsStaff() && ticket.Requester.ID != claims.UserID() {
		cleanup()
		writeError(w, http.StatusForbidden, "no puedes adjuntar archivos a este ticket")
		return
	}

	var id int64
	err = s.db.QueryRow(r.Context(), `
		INSERT INTO attachments (ticket_id, comment_id, uploader_id, filename, content_type, size, storage_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`, ticketID, commentID, claims.UserID(), filename, contentType, size, key,
	).Scan(&id)
	if err != nil {
		cleanup()
		internalError(w, err)
		return
	}
	a, err := scanAttachment(s.db.QueryRow(r.Context(), attachmentSelect+" WHERE a.id = $1", id))
	if err != nil {
		internalError(w, err)
		return
	}
	var out fanout
	out.ticket(ticket, a.Internal)
	s.broker.publish(out)
	writeJSON(w, http.StatusCreated, a)
}

func uploadReadError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeError(w, http.StatusRequestEntityTooLarge, "el archivo es demasiado grande")
		return
	}
	writeError(w, http.StatusBadRequest, "no se pudo leer el archivo")
}

// store guarda el contenido en disco con un nombre aleatorio y detecta su tipo real.
// Lee como máximo maxUpload+1 bytes: si devuelve más que maxUpload, el archivo es demasiado grande.
func (s *Server) store(src io.Reader) (key, contentType string, size int64, err error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", "", 0, err
	}
	key = hex.EncodeToString(raw[:])
	path := s.storagePath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", "", 0, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return "", "", 0, err
	}
	defer f.Close()

	head := make([]byte, 512)
	n, err := io.ReadFull(src, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		os.Remove(path)
		return "", "", 0, err
	}
	head = head[:n]
	contentType = http.DetectContentType(head)
	if _, err := f.Write(head); err != nil {
		os.Remove(path)
		return "", "", 0, err
	}
	rest, err := io.Copy(f, io.LimitReader(src, s.maxUpload+1-int64(n)))
	if err != nil {
		os.Remove(path)
		return "", "", 0, err
	}
	return key, contentType, int64(n) + rest, f.Close()
}

func (s *Server) storagePath(key string) string {
	return filepath.Join(s.uploadDir, key[:2], key)
}

// cleanFilename deja solo el nombre (sin rutas) y quita caracteres de control.
func cleanFilename(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, `\`, "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '"' {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == "/" {
		name = "archivo"
	}
	if r := []rune(name); len(r) > 200 {
		ext := filepath.Ext(name)
		name = string(r[:200-len([]rune(ext))]) + ext
	}
	return name
}

// loadAttachment devuelve el adjunto si el usuario puede verlo.
func (s *Server) loadAttachment(w http.ResponseWriter, r *http.Request) (Attachment, bool) {
	id, ok := pathID(w, r)
	if !ok {
		return Attachment{}, false
	}
	a, err := scanAttachment(s.db.QueryRow(r.Context(), attachmentSelect+" WHERE a.id = $1", id))
	claims := claimsFrom(r)
	if err == nil {
		var requester, orgID int64
		err = s.db.QueryRow(r.Context(), `SELECT requester_id, org_id FROM tickets WHERE id = $1`, a.TicketID).
			Scan(&requester, &orgID)
		// Otra empresa, o un cliente que no es el solicitante o mira una nota interna: no existe.
		if err == nil && (orgID != claims.OrgID || (!claims.IsStaff() && (requester != claims.UserID() || a.Internal))) {
			err = pgx.ErrNoRows
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "archivo no encontrado")
		return Attachment{}, false
	}
	if err != nil {
		internalError(w, err)
		return Attachment{}, false
	}
	return a, true
}

// inlineTypes se pueden mostrar en el navegador; el resto se sirve como descarga genérica.
var inlineTypes = []string{"image/png", "image/jpeg", "image/gif", "image/webp", "application/pdf"}

func (s *Server) downloadAttachment(w http.ResponseWriter, r *http.Request) {
	a, ok := s.loadAttachment(w, r)
	if !ok {
		return
	}
	f, err := os.Open(s.storagePath(a.storageKey))
	if err != nil {
		slog.Error("adjunto sin archivo en disco", "id", a.ID, "error", err)
		writeError(w, http.StatusNotFound, "el archivo ya no está disponible")
		return
	}
	defer f.Close()
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(5 * time.Minute))

	contentType := "application/octet-stream"
	if slices.Contains(inlineTypes, a.ContentType) {
		contentType = a.ContentType
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(a.Filename))
	http.ServeContent(w, r, "", a.CreatedAt, f)
}

// deleteAttachment lo puede borrar quien lo subió o un administrador.
func (s *Server) deleteAttachment(w http.ResponseWriter, r *http.Request) {
	a, ok := s.loadAttachment(w, r)
	if !ok {
		return
	}
	claims := claimsFrom(r)
	if a.Uploader.ID != claims.UserID() && claims.Role != "admin" {
		writeError(w, http.StatusForbidden, "solo quien lo subió puede borrarlo")
		return
	}
	if _, err := s.db.Exec(r.Context(), `DELETE FROM attachments WHERE id = $1`, a.ID); err != nil {
		internalError(w, err)
		return
	}
	if err := os.Remove(s.storagePath(a.storageKey)); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("no se pudo borrar el archivo", "key", a.storageKey, "error", err)
	}
	w.WriteHeader(http.StatusNoContent)
}
