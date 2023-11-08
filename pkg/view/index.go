package view

import (
	"html/template"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/Kelvedler/ChemicalStorage/pkg/db"
)

func Index(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	storageUser, _ := db.StorageUserGetByID(r.Context(), rc.dbpool, rc.userID)
	data := userData{
		User: storageUser,
	}
	tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/base.html"))
	tmpl.Execute(w, data)
}
