package view

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func Favicon(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	http.ServeFile(w, r, "static/favicon.png")
}
