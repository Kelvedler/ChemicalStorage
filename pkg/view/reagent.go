package view

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/go-playground/validator/v10"
	"github.com/julienschmidt/httprouter"

	"github.com/Kelvedler/ChemicalStorage/pkg/common"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
)

type reagentsData struct {
	ReagentsSlice []db.ReagentFull
}

type reagentData struct {
	Reagent db.ReagentFull
}

func (handlerContext *HandlerContext) Reagents(
	rc RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	reagentsSlice, err := db.ReagentGetRange(r.Context(), handlerContext.dbpool, 20, 0, "")
	if err != nil {
		rc.logger.Error(err.Error())
		common.DefaultErrorResp(w)
		return
	}
	data := reagentsData{
		ReagentsSlice: reagentsSlice,
	}
	tmpl := template.Must(
		template.ParseFiles(
			"templates/reagents.html",
			"templates/base.html",
			"templates/reagents-assets.html",
		),
	)
	tmpl.Execute(w, data)
}

func (handlerContext *HandlerContext) Reagent(
	rc RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	reagent_id := params.ByName("id")
	reagent, err := db.ReagentGet(r.Context(), handlerContext.dbpool, reagent_id)
	if err != nil {
		errStruct := db.ErrorAsStruct(err)
		switch errStruct.(type) {
		case db.InvalidUUID, db.DoesNotExist:
			rc.logger.Info("Not found")
			common.ErrorRespNotFound(w)
			return
		default:
			panic(fmt.Sprintf("unexpected err type, %t", errStruct))
		}
	}
	data := reagentData{
		Reagent: reagent,
	}
	tmpl := template.Must(template.ParseFiles("templates/reagent.html", "templates/base.html"))
	tmpl.Execute(w, data)
}

func (handlerContext *HandlerContext) ReagentCreate(
	rc RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	tmpl := template.Must(
		template.ParseFiles(
			"templates/reagent-new.html",
			"templates/reagents-assets.html",
			"templates/base.html",
		),
	)
	tmpl.Execute(w, nil)
}

func (handlerContext *HandlerContext) ReagentsAPI(
	rc RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	tmpl := template.Must(
		template.ParseFiles(
			"templates/reagents-assets.html",
		),
	).Lookup("reagents-search")

	srcForm := common.SrcForm{Src: r.URL.Query().Get("src")}
	err := handlerContext.validate.Struct(srcForm)
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), srcForm)
		rc.logger.Info(err.Error())
		tmpl.Execute(w, nil)
		return
	}
	reagentsSlice, err := db.ReagentGetRange(r.Context(), handlerContext.dbpool, 20, 0, srcForm.Src)
	if err != nil {
		rc.logger.Error(err.Error())
		tmpl.Execute(w, nil)
		return
	}
	data := reagentsData{
		ReagentsSlice: reagentsSlice,
	}
	tmpl.Execute(w, data)
}

func sanitizeReagentShort(handlerContext *HandlerContext, reagent *db.ReagentShort) {
	sanitizer := handlerContext.sanitize
	reagent.Name = sanitizer.Sanitize(reagent.Name)
	reagent.Formula = sanitizer.Sanitize(reagent.Formula)
}

func (handlerContext *HandlerContext) ReagentCreateAPI(
	rc RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	var reagent db.ReagentShort
	err := common.BindJSON(r, &reagent)
	if err != nil {
		rc.logger.Error(err.Error())
		common.DefaultErrorResp(w)
		return
	}

	sanitizeReagentShort(handlerContext, &reagent)

	err = handlerContext.validate.Struct(reagent)
	if err != nil {
		tmpl := template.Must(template.ParseFiles("templates/reagents-assets.html")).
			Lookup("create-form")
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), reagent)
		rc.logger.Info(err.Error())
		errMap := err.(common.ValidationError).Map()
		errMap["Formula"] = reagent.Formula
		errMap["Name"] = reagent.Name
		tmpl.Execute(w, errMap)
		return
	}
	reargentNew, err := db.ReagentCreate(r.Context(), handlerContext.dbpool, reagent)
	if err != nil {
		errStruct := db.ErrorAsStruct(err)
		switch errStruct.(type) {
		case db.UniqueViolation:
			err = errStruct.(db.UniqueViolation).LocalizeUniqueViolation(db.ReagentShort{})
			rc.logger.Info(err.Error())
			tmpl := template.Must(template.ParseFiles("templates/reagents-assets.html")).
				Lookup("create-form")
			errMap := err.(db.DBError).Map()
			errMap["Formula"] = reagent.Formula
			errMap["Name"] = reagent.Name
			tmpl.Execute(w, errMap)
			return
		default:
			panic(fmt.Sprintf("unexpected err type, %t", errStruct))
		}
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/reagents/%s", reargentNew.ID))
}
