package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"golang.org/x/net/xsrftoken"

	"github.com/Kelvedler/ChemicalStorage/pkg/env"
)

func ValidateForXSRF(r *http.Request, userID uuid.UUID) bool {
	if r.Method == http.MethodGet {
		return true
	}
	token := r.Header.Get("_xsrf")
	return xsrftoken.Valid(token, env.Env.SecretKey, userID.String(), r.URL.String())
}
