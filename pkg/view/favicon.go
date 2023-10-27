package view

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (handler *BaseHandler) Favicon(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	http.ServeFile(w, r, "static/favicon.png")
}
