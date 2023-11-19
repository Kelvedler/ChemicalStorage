package view

import (
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/xsrftoken"

	"github.com/Kelvedler/ChemicalStorage/pkg/common"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
	"github.com/Kelvedler/ChemicalStorage/pkg/env"
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
	Caller     db.StorageUser
	ID         string
	Name       string
	Formula    string
	NameErr    string
	FormulaErr string
	PostXsrf   string
	PutXsrf    string
}

func getReagentPostXsrf(userID string) string {
	return xsrftoken.Generate(
		env.Env.SecretKey,
		userID,
		"/api/v1/reagents",
	)
}

func getReagentPutXsrf(userID string, reagentID uuid.UUID) string {
	return xsrftoken.Generate(
		env.Env.SecretKey,
		userID,
		fmt.Sprintf("/api/v1/reagents/%s", reagentID),
	)
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
	reagentID := params.ByName("id")
	reagent, err := db.ReagentGet(r.Context(), rc.dbpool, reagentID)
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
		Caller:  storageUser,
		ID:      reagentID,
		Name:    reagent.Name,
		Formula: reagent.Formula,
	}
	tmpl := template.Must(
		template.ParseFiles(
			"templates/reagent.html",
			"templates/reagents-assets.html",
			"templates/base.html",
		),
	)
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
	data := reagentData{
		Caller:   storageUser,
		PostXsrf: getReagentPostXsrf(rc.userID),
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

func reagentErrMapAddInput(errMap map[string]string, reagent db.Reagent) {
	errMap["Formula"] = reagent.Formula
	errMap["Name"] = reagent.Name
}

func reagentPostErrMapAddInput(errMap map[string]string, reagent db.Reagent, userID string) {
	reagentErrMapAddInput(errMap, reagent)
	errMap["PostXsrf"] = getReagentPostXsrf(userID)
}

func reagentPutErrMapAddInput(
	errMap map[string]string,
	reagent db.Reagent,
	userID string,
	reagentID uuid.UUID,
) {
	reagentErrMapAddInput(errMap, reagent)
	errMap["PutXsrf"] = getReagentPutXsrf(userID, reagentID)
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
		Lookup("reagent-form")

	err = rc.validate.StructPartial(reagent, "Name", "Formula")
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), reagent)
		rc.logger.Info(err.Error())
		errMap := err.(common.ValidationError).Map()
		reagentPostErrMapAddInput(errMap, reagent, rc.userID)
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
			reagentPostErrMapAddInput(errMap, reagent, rc.userID)
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

func ReagentPutAPI(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	var reagent db.Reagent
	err := common.BindJSON(r, &reagent)
	if err != nil {
		rc.logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	sanitizeReagent(rc, &reagent)
	tmpl := template.Must(template.ParseFiles("templates/reagents-assets.html"))

	errTmpl := tmpl.Lookup("reagent-form")

	err = rc.validate.StructPartial(reagent, "Name", "Formula")
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), reagent)
		rc.logger.Info(err.Error())
		errMap := err.(common.ValidationError).Map()
		reagentPutErrMapAddInput(errMap, reagent, rc.userID, reagent.ID)
		errTmpl.Execute(w, errMap)
		return
	}
	reagent.ID, err = uuid.Parse(params.ByName("id"))
	if err != nil {
		rc.logger.Info("Not found")
		common.ErrorResp(w, common.NotFound)
		return
	}
	err = reagent.ReagentUpdate(r.Context(), rc.dbpool)
	if err != nil {
		errStruct := db.ErrorAsStruct(err)
		switch errStruct.(type) {
		case db.UniqueViolation:
			err = errStruct.(db.UniqueViolation).LocalizeUniqueViolation(db.Reagent{})
			rc.logger.Info(err.Error())
			errMap := err.(db.DBError).Map()
			reagentPostErrMapAddInput(errMap, reagent, rc.userID)
			errTmpl.Execute(w, errMap)
			return
		default:
			rc.logger.Error(err.Error())
			common.ErrorResp(w, common.Internal)
			return
		}
	}
	successTmpl := tmpl.Lookup("reagent")
	caller := db.StorageUser{
		Role: rc.userRole,
	}
	data := reagentData{
		Caller:  caller,
		ID:      reagent.ID.String(),
		Name:    reagent.Name,
		Formula: reagent.Formula,
	}
	w.Header().Set("HX-Retarget", "#reagent")
	successTmpl.Execute(w, data)
}

func ReagentEditFormAPI(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	var reagent db.Reagent
	err := common.BindJSON(r, &reagent)
	if err != nil {
		rc.logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	reagent.ID, err = uuid.Parse(params.ByName("id"))
	if err != nil {
		rc.logger.Info("Not found")
		common.ErrorResp(w, common.NotFound)
		return
	}

	tmpl := template.Must(template.ParseFiles("templates/reagents-assets.html")).
		Lookup("reagent-edit")
	caller := db.StorageUser{
		Role: rc.userRole,
	}
	data := reagentData{
		Caller:  caller,
		ID:      reagent.ID.String(),
		Name:    reagent.Name,
		Formula: reagent.Formula,
		PutXsrf: getReagentPutXsrf(rc.userID, reagent.ID),
	}
	tmpl.Execute(w, data)
}

func ReagentReadFormAPI(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	reagentID := params.ByName("id")
	reagent, err := db.ReagentGet(r.Context(), rc.dbpool, reagentID)
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
	tmpl := template.Must(template.ParseFiles("templates/reagents-assets.html")).
		Lookup("reagent")
	caller := db.StorageUser{
		Role: rc.userRole,
	}
	data := reagentData{
		Caller:  caller,
		ID:      reagentID,
		Name:    reagent.Name,
		Formula: reagent.Formula,
	}
	tmpl.Execute(w, data)
}
