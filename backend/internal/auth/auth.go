// Package auth maneja contraseñas y tokens JWT.
package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	RoleAdmin    = "admin"
	RoleAgent    = "agent"
	RoleCustomer = "customer"
)

// Claims es lo que viaja dentro del token.
type Claims struct {
	Role string `json:"role"`
	// OrgID no viaja en el token: la API lo lee de la base de datos en cada petición.
	OrgID int64 `json:"-"`
	jwt.RegisteredClaims
}

// UserID devuelve el id del usuario guardado en el campo "sub".
func (c Claims) UserID() int64 {
	id, _ := strconv.ParseInt(c.Subject, 10, 64)
	return id
}

// IsStaff indica si el usuario es agente o administrador.
func (c Claims) IsStaff() bool {
	return c.Role == RoleAgent || c.Role == RoleAdmin
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

type Issuer struct {
	secret []byte
	ttl    time.Duration
}

func NewIssuer(secret []byte, ttl time.Duration) *Issuer {
	return &Issuer{secret: secret, ttl: ttl}
}

func (i *Issuer) Issue(userID int64, role string) (string, time.Time, error) {
	expires := time.Now().Add(i.ttl)
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expires),
			Issuer:    "micolmena",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
	return token, expires, err
}

func (i *Issuer) Parse(token string) (Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) {
		return i.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer("micolmena"),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return Claims{}, err
	}
	if claims.UserID() == 0 {
		return Claims{}, errors.New("token sin usuario")
	}
	return claims, nil
}
