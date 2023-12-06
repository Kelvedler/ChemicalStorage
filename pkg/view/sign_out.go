package view

import (
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/Kelvedler/ChemicalStorage/pkg/auth"
	"github.com/Kelvedler/ChemicalStorage/pkg/middleware"
)

func SignOutAPI(
	_ *middleware.RequestContext,
	w http.ResponseWriter,
	_ *http.Request,
	_ httprouter.Params,
) {
	auth.SetEmptyTokenCookie(w)
	w.Header().Add("HX-Refresh", "true")
}
