package view

import (
	"html/template"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (handler *BaseHandler) Index(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/base.html"))
	tmpl.Execute(w, nil)
}
