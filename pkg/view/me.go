package view

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/Kelvedler/ChemicalStorage/pkg/common"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
)

func Me(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	storageUser, err := db.StorageUserGetByID(r.Context(), rc.dbpool, rc.userID)
	if err != nil {
		errStruct := db.ErrorAsStruct(err)
		switch errStruct.(type) {
		case db.InvalidUUID, db.DoesNotExist:
			rc.logger.Info("Not found")
			common.ErrorResp(w, common.NotFound)
			return
		default:
			panic(fmt.Sprintf("unexpected err type, %t", errStruct))
		}
	}
	data := CallerData{
		Caller: storageUser,
	}
	tmpl := template.Must(template.ParseFiles("templates/me.html", "templates/base.html"))
	tmpl.Execute(w, data)
}
