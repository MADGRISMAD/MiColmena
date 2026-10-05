package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

const maxBodyBytes = 1 << 20 // 1 MB

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("escribir respuesta", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func internalError(w http.ResponseWriter, err error) {
	slog.Error("error interno", "error", err)
	writeError(w, http.StatusInternalServerError, "error interno")
}

// decodeJSON lee el cuerpo como JSON, rechazando campos desconocidos y cuerpos muy grandes.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "el cuerpo es demasiado grande")
		} else {
			writeError(w, http.StatusBadRequest, "JSON inválido: "+err.Error())
		}
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id inválido")
		return 0, false
	}
	return id, true
}

// validationErrors acumula errores de validación por campo.
type validationErrors map[string]string

func (v validationErrors) check(ok bool, field, msg string) {
	if !ok {
		if _, exists := v[field]; !exists {
			v[field] = msg
		}
	}
}

func (v validationErrors) write(w http.ResponseWriter) bool {
	if len(v) == 0 {
		return false
	}
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"error":  "datos inválidos",
		"fields": v,
	})
	return true
}

func notBlank(s string) bool { return strings.TrimSpace(s) != "" }
