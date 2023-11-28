package view

import (
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
	caller := db.StorageUser{ID: rc.userID}
	errs := db.PerformBatch(r.Context(), rc.dbpool, []db.BatchSet{caller.GetByID})
	userErr := errs[0]
	if userErr != nil {
		errStruct := db.ErrorAsStruct(userErr)
		switch errStruct.(type) {
		case db.InvalidUUID, db.DoesNotExist:
			rc.logger.Info("Not found")
			common.ErrorResp(w, common.NotFound)
			return
		default:
			rc.logger.Error(userErr.Error())
			common.ErrorResp(w, common.Internal)
			return
		}
	}
	data := CallerData{
		Caller: caller,
	}
	tmpl := template.Must(template.ParseFiles("templates/me.html", "templates/base.html"))
	tmpl.Execute(w, data)
}
