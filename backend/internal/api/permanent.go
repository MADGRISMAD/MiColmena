package api

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/MADGRISMAD/MiColmena/backend/internal/auth"
)

// permanentAdmins son administradores que nadie puede desactivar, bajar de rol ni cambiarles el
// email. Es la misma lista en MiColmena, MiTiendita y MiConsultorio; para cambiarla hay que
// cambiar el código, así queda en el historial de git.
var permanentAdmins = []string{"madgrismad@gmail.com", "mayra.bamaca09@gmail.com", "luispantoja1102@gmail.com"}

// isPermanentAdmin exige además el rol: quien se registre como cliente con uno de esos correos
// no queda protegido.
func isPermanentAdmin(u User) bool {
	return u.Role == auth.RoleAdmin && slices.Contains(permanentAdmins, strings.ToLower(strings.TrimSpace(u.Email)))
}

// permanentChange dice por qué no se permite un cambio sobre un administrador permanente, o "" si
// se permite. Su contraseña solo la cambia otro permanente: si no, cualquiera se quedaría con la cuenta.
func (s *Server) permanentChange(ctx context.Context, actorID, targetID int64, role *string, active *bool, email, password *string) (string, error) {
	if actorID == targetID {
		return "", nil
	}
	target, err := scanUser(s.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, targetID))
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil // el handler responde 404
	}
	if err != nil || !target.Permanent {
		return "", err
	}
	switch {
	case role != nil && *role != auth.RoleAdmin:
		return "es un administrador permanente: no se le puede quitar el rol", nil
	case active != nil && !*active:
		return "es un administrador permanente: no se puede desactivar", nil
	case email != nil && !strings.EqualFold(strings.TrimSpace(*email), target.Email):
		return "es un administrador permanente: no se le puede cambiar el email", nil
	case password != nil:
		actor, err := scanUser(s.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, actorID))
		if err != nil {
			return "", err
		}
		if !actor.Permanent {
			return "solo otro administrador permanente puede cambiarle la contraseña", nil
		}
	}
	return "", nil
}
