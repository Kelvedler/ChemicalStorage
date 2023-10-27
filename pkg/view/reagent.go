package view

import (
	"fmt"
	"net/http"
	"text/template"

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

func (handler *BaseHandler) Reagents(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	reagentsSlice, err := db.ReagentGetRange(r.Context(), handler.dbpool, 20, 0, "")
	if err != nil {
		fmt.Println(err)
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

func (handler *BaseHandler) Reagent(
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	reagent_id := params.ByName("id")
	reagent, err := db.ReagentGet(r.Context(), handler.dbpool, reagent_id)
	if err != nil {
		fmt.Println(err)
		return
	}
	data := reagentData{
		Reagent: reagent,
	}
	tmpl := template.Must(template.ParseFiles("templates/reagent.html", "templates/base.html"))
	tmpl.Execute(w, data)
}

func (handler *BaseHandler) ReagentCreate(
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

func (handler *BaseHandler) ReagentsAPI(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	srcForm := common.SrcForm{Src: r.URL.Query().Get("src")}
	err := handler.validate.Struct(srcForm)
	if err != nil {
		fmt.Println(err)
	}
	reagentsSlice, err := db.ReagentGetRange(r.Context(), handler.dbpool, 20, 0, srcForm.Src)
	if err != nil {
		fmt.Println(err)
	}
	data := reagentsData{
		ReagentsSlice: reagentsSlice,
	}
	tmpl := template.Must(
		template.ParseFiles(
			"templates/reagents-assets.html",
		),
	).Lookup("reagents-search")
	tmpl.Execute(w, data)
}

func (handler *BaseHandler) ReagentCreateAPI(
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	var reagent db.ReagentShort
	err := common.BindJSON(r, &reagent)
	if err != nil {
		common.DefaultErrorResp(w)
		return
	}
	err = common.ValidateStruct(handler.validate, reagent)
	if err != nil {
		tmpl := template.Must(template.ParseFiles("templates/reagents-assets.html")).
			Lookup("create-form")
		errMap := err.(common.ValidationError).Map()
		errMap["Formula"] = reagent.Formula
		errMap["Name"] = reagent.Name
		tmpl.Execute(w, errMap)
		return
	}
	reargentNew, err := db.ReagentCreate(r.Context(), handler.dbpool, reagent)
	if err != nil {
		err = db.LocalizeError(err, db.ReagentShort{})
		tmpl := template.Must(template.ParseFiles("templates/reagents-assets.html")).
			Lookup("create-form")
		errMap := err.(db.DBError).Map()
		errMap["Formula"] = reagent.Formula
		errMap["Name"] = reagent.Name
		tmpl.Execute(w, errMap)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/reagents/%s", reargentNew.ID))
}
