package view

import (
	"html/template"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/Kelvedler/ChemicalStorage/pkg/common"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
	"github.com/Kelvedler/ChemicalStorage/pkg/middleware"
)

func Me(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	caller := db.StorageUser{ID: rc.UserID}
	errs := db.PerformBatch(r.Context(), rc.DBpool, []db.BatchSet{caller.GetByID})
	userErr := errs[0]
	if userErr != nil {
		errStruct := db.ErrorAsStruct(userErr)
		switch errStruct.(type) {
		case db.InvalidUUID, db.DoesNotExist:
			rc.Logger.Info("Not found")
			common.ErrorResp(w, common.NotFound)
			return
		default:
			rc.Logger.Error(userErr.Error())
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
