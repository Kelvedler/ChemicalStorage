package auth

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"

	"github.com/Kelvedler/ChemicalStorage/pkg/db"
	"github.com/Kelvedler/ChemicalStorage/pkg/env"
)

const cookiePath = "/"

func setTokenCookie(w http.ResponseWriter, token string) error {
	jwtEnv := env.Env.Jwt
	cookie := http.Cookie{
		Name:     "access",
		Value:    token,
		Path:     cookiePath,
		Domain:   jwtEnv.Domain,
		Secure:   jwtEnv.SecureCookies,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  JWTExpiration(),
	}
	http.SetCookie(w, &cookie)
	return nil
}

func SetNewTokenCookie(w http.ResponseWriter, user db.StorageUser) error {
	token, err := IssueJWT(user)
	if err != nil {
		return err
	}
	return setTokenCookie(w, token)
}

func ReissueTokenCookie(w http.ResponseWriter, claims jwt.MapClaims) error {
	token, err := ReissueJWT(claims)
	if err != nil {
		return err
	}
	return setTokenCookie(w, token)
}

func SetEmptyTokenCookie(w http.ResponseWriter) {
	cookie := http.Cookie{
		Name:    "access",
		Value:   "",
		Path:    cookiePath,
		Expires: time.Unix(0, 0),
	}
	http.SetCookie(w, &cookie)
}
