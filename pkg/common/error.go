package common

import (
	"net/http"
	"text/template"
)

func DefaultErrorResp(w http.ResponseWriter) {
	w.Header().Set("HX-Retarget", "#content")
	tmpl := template.Must(template.ParseFiles("templates/base.html")).Lookup("error-page")
	tmpl.Execute(w, nil)
}
