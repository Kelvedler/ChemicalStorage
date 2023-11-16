package view

import (
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/go-playground/validator/v10"
	"github.com/julienschmidt/httprouter"

	"github.com/Kelvedler/ChemicalStorage/pkg/common"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
)

type reagentsData struct {
	ReagentsSlice []db.Reagent
	LastReagent   db.Reagent
	NextOffset    int
	Src           string
	Caller        db.StorageUser
}

func newReagentsData(reagentsSlice []db.Reagent, src string, offset int) (data reagentsData) {
	if len(reagentsSlice) > 1 {
		data.ReagentsSlice = reagentsSlice[:len(reagentsSlice)-1]
		data.LastReagent = reagentsSlice[len(reagentsSlice)-1]
		data.NextOffset = offset + len(reagentsSlice)
		data.Src = src
	} else {
		data.ReagentsSlice = reagentsSlice
	}
	return data
}

type reagentData struct {
	Reagent db.Reagent
	Caller  db.StorageUser
}

type reagentNewData struct {
	Caller     db.StorageUser
	Name       string
	Formula    string
	NameErr    string
	FormulaErr string
}

func Reagents(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	offset := 0
	src := ""
	reagentsSlice, err := db.ReagentGetRange(r.Context(), rc.dbpool, 20, offset, src)
	if err != nil {
		rc.logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	data := newReagentsData(reagentsSlice, src, offset)
	storageUser, _ := db.StorageUserGetByID(r.Context(), rc.dbpool, rc.userID)
	data.Caller = storageUser
	tmpl := template.Must(
		template.ParseFiles(
			"templates/reagents.html",
			"templates/base.html",
			"templates/reagents-assets.html",
		),
	)
	tmpl.Execute(w, data)
}

func Reagent(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	reagent_id := params.ByName("id")
	reagent, err := db.ReagentGet(r.Context(), rc.dbpool, reagent_id)
	if err != nil {
		errStruct := db.ErrorAsStruct(err)
		switch errStruct.(type) {
		case db.InvalidUUID, db.DoesNotExist:
			rc.logger.Info("Not found")
			common.ErrorResp(w, common.NotFound)
			return
		default:
			rc.logger.Error(err.Error())
			common.ErrorResp(w, common.Internal)
			return
		}
	}
	storageUser, _ := db.StorageUserGetByID(r.Context(), rc.dbpool, rc.userID)
	data := reagentData{
		Reagent: reagent,
		Caller:  storageUser,
	}
	tmpl := template.Must(template.ParseFiles("templates/reagent.html", "templates/base.html"))
	tmpl.Execute(w, data)
}

func ReagentCreate(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	tmpl := template.Must(
		template.ParseFiles(
			"templates/reagent-new.html",
			"templates/base.html",
			"templates/reagents-assets.html",
		),
	)
	storageUser, _ := db.StorageUserGetByID(r.Context(), rc.dbpool, rc.userID)
	data := reagentNewData{
		Caller: storageUser,
	}
	tmpl.Execute(w, data)
}

type ReagentsAPIForm struct {
	Src    string `json:"src"    validate:"omitempty,lte=50"          uaLocal:"пошук"`
	Offset int    `json:"offset" validate:"omitempty,min=0,max=10000" uaLocal:"зміщення"`
}

func ReagentsAPI(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	src := r.URL.Query().Get("src")
	offsetStr := r.URL.Query().Get("offset")
	if offsetStr == "" {
		offsetStr = "0"
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		rc.logger.Info(err.Error())
		return
	}

	srcForm := ReagentsAPIForm{Src: src, Offset: offset}
	err = rc.validate.Struct(srcForm)
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), srcForm)
		rc.logger.Info(err.Error())
		w.WriteHeader(400)
		return
	}
	reagentsSlice, err := db.ReagentGetRange(
		r.Context(),
		rc.dbpool,
		20,
		offset,
		srcForm.Src,
	)
	if err != nil {
		rc.logger.Error(err.Error())
		w.WriteHeader(400)
		return
	}
	data := newReagentsData(reagentsSlice, src, offset)
	tmpl := template.Must(
		template.ParseFiles(
			"templates/reagents.html",
			"templates/reagents-assets.html",
		),
	).Lookup("reagents-search")
	tmpl.Execute(w, data)
}

func sanitizeReagent(rc *RequestContext, reagent *db.Reagent) {
	sanitizer := rc.sanitize
	reagent.Name = sanitizer.Sanitize(reagent.Name)
	reagent.Formula = sanitizer.Sanitize(reagent.Formula)
}

func ReagentCreateAPI(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	var reagent db.Reagent
	err := common.BindJSON(r, &reagent)
	if err != nil {
		rc.logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}

	sanitizeReagent(rc, &reagent)
	tmpl := template.Must(template.ParseFiles("templates/reagents-assets.html")).
		Lookup("create-form")

	err = rc.validate.StructPartial(reagent, "Name", "Formula")
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), reagent)
		rc.logger.Info(err.Error())
		errMap := err.(common.ValidationError).Map()
		errMap["Formula"] = reagent.Formula
		errMap["Name"] = reagent.Name
		tmpl.Execute(w, errMap)
		return
	}
	reargentNew, err := db.ReagentCreate(r.Context(), rc.dbpool, reagent)
	if err != nil {
		errStruct := db.ErrorAsStruct(err)
		switch errStruct.(type) {
		case db.UniqueViolation:
			err = errStruct.(db.UniqueViolation).LocalizeUniqueViolation(db.Reagent{})
			rc.logger.Info(err.Error())
			errMap := err.(db.DBError).Map()
			errMap["Formula"] = reagent.Formula
			errMap["Name"] = reagent.Name
			tmpl.Execute(w, errMap)
			return
		default:
			rc.logger.Error(err.Error())
			common.ErrorResp(w, common.Internal)
			return
		}
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/reagents/%s", reargentNew.ID))
}
