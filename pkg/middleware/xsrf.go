package middleware

import (
	"net/http"

	"golang.org/x/net/xsrftoken"

	"github.com/Kelvedler/ChemicalStorage/pkg/env"
)

func ValidateForXSRF(r *http.Request, userID string) bool {
	if r.Method == http.MethodGet {
		return true
	}
	token := r.Header.Get("_xsrf")
	return xsrftoken.Valid(token, env.Env.SecretKey, userID, r.URL.String())
}
