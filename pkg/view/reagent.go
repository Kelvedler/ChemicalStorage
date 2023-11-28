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

func (data *reagentsData) set(reagentsSlice []db.Reagent, src string, offset int) {
	if len(reagentsSlice) > 1 {
		data.ReagentsSlice = reagentsSlice[:len(reagentsSlice)-1]
		data.LastReagent = reagentsSlice[len(reagentsSlice)-1]
		data.NextOffset = offset + len(reagentsSlice)
		data.Src = src
	} else {
		data.ReagentsSlice = reagentsSlice
	}
}

type reagentData struct {
	Caller         db.StorageUser
	ID             string
	Name           string
	Formula        string
	NameErr        string
	FormulaErr     string
	PostXsrf       string
	PutXsrf        string
	InstancesSlice []db.ReagentInstance
	LastInstance   db.ReagentInstance
	NextOffset     int
}

func (data *reagentData) addInstances(instancesSlice []db.ReagentInstance, offset int) {
	if len(instancesSlice) > 1 {
		data.InstancesSlice = instancesSlice[:len(instancesSlice)-1]
		data.LastInstance = instancesSlice[len(instancesSlice)-1]
		data.NextOffset = offset + len(instancesSlice)
	} else {
		data.InstancesSlice = instancesSlice
	}
}

func getReagentPostXsrf(userID uuid.UUID) string {
	return xsrftoken.Generate(
		env.Env.SecretKey,
		userID.String(),
		"/api/v1/reagents",
	)
}

func getReagentPutXsrf(userID, reagentID uuid.UUID) string {
	return xsrftoken.Generate(
		env.Env.SecretKey,
		userID.String(),
		fmt.Sprintf("/api/v1/reagents/%s", reagentID),
	)
}

func Reagents(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	src := ""
	offset := 0
	reagentsRange := db.ReagentsRange{
		Limit:  20,
		Offset: offset,
		Src:    src,
	}
	caller := db.StorageUser{ID: rc.userID}
	errs := db.PerformBatch(
		r.Context(),
		rc.dbpool,
		[]db.BatchSet{reagentsRange.Get, caller.GetByID},
	)
	reagentsErr := errs[0]
	if reagentsErr != nil {
		rc.logger.Error(reagentsErr.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	data := reagentsData{Caller: caller}
	data.set(reagentsRange.Reagents, src, offset)
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
	offset := 0
	reagentID, err := uuid.Parse(params.ByName("reagentID"))
	if err != nil {
		rc.logger.Info("Invalid UUID")
		common.ErrorResp(w, common.NotFound)
		return
	}

	reagent := db.Reagent{ID: reagentID}
	rir := db.ReagentInstanceRange{
		ReagentID: reagentID,
		Limit:     20,
		Offset:    offset,
	}
	caller := db.StorageUser{ID: rc.userID}
	errs := db.PerformBatch(
		r.Context(),
		rc.dbpool,
		[]db.BatchSet{reagent.Get, rir.Get, caller.GetByID},
	)
	reagentErr := errs[0]
	reagentInstanceErr := errs[1]
	if reagentErr != nil {
		errStruct := db.ErrorAsStruct(reagentErr)
		switch errStruct.(type) {
		case db.DoesNotExist:
			rc.logger.Info("Not found")
			common.ErrorResp(w, common.NotFound)
			return
		default:
			rc.logger.Error(reagentErr.Error())
			common.ErrorResp(w, common.Internal)
			return
		}
	}
	if reagentInstanceErr != nil {
		rc.logger.Error(reagentInstanceErr.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	data := reagentData{
		Caller:  caller,
		ID:      reagent.ID.String(),
		Name:    reagent.Name,
		Formula: reagent.Formula,
	}
	data.addInstances(rir.ReagentInstances, offset)
	tmpl := template.Must(
		template.ParseFiles(
			"templates/reagent.html",
			"templates/reagents-assets.html",
			"templates/instances-assets.html",
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
	caller := db.StorageUser{ID: rc.userID}
	_ = db.PerformBatch(r.Context(), rc.dbpool, []db.BatchSet{caller.GetByID})
	data := reagentData{
		Caller:   caller,
		PostXsrf: getReagentPostXsrf(rc.userID),
	}
	tmpl.Execute(w, data)
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

	searchForm := SearchAPIForm{Src: src, Offset: offset}
	err = rc.validate.Struct(searchForm)
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), searchForm)
		rc.logger.Info(err.Error())
		w.WriteHeader(400)
		return
	}
	reagentsRange := db.ReagentsRange{
		Limit:  20,
		Offset: offset,
		Src:    searchForm.Src,
	}
	errs := db.PerformBatch(r.Context(), rc.dbpool, []db.BatchSet{reagentsRange.Get})
	reagentsErr := errs[0]
	if reagentsErr != nil {
		rc.logger.Error(reagentsErr.Error())
		w.WriteHeader(400)
		return
	}
	var data reagentsData
	data.set(reagentsRange.Reagents, src, offset)
	tmpl := template.Must(
		template.ParseFiles(
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

func reagentPostErrMapAddInput(errMap map[string]string, reagent db.Reagent, userID uuid.UUID) {
	reagentErrMapAddInput(errMap, reagent)
	errMap["PostXsrf"] = getReagentPostXsrf(userID)
}

func reagentPutErrMapAddInput(
	errMap map[string]string,
	reagent db.Reagent,
	userID uuid.UUID,
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
	errs := db.PerformBatch(r.Context(), rc.dbpool, []db.BatchSet{reagent.Create})
	reagentErr := errs[0]
	if reagentErr != nil {
		errStruct := db.ErrorAsStruct(reagentErr)
		switch errStruct.(type) {
		case db.UniqueViolation:
			err = errStruct.(db.UniqueViolation).LocalizeUniqueViolation(db.Reagent{})
			rc.logger.Info(err.Error())
			errMap := err.(db.DBError).Map()
			reagentPostErrMapAddInput(errMap, reagent, rc.userID)
			tmpl.Execute(w, errMap)
			return
		default:
			rc.logger.Error(reagentErr.Error())
			common.ErrorResp(w, common.Internal)
			return
		}
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/reagents/%s", reagent.ID))
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
	reagent.ID, err = uuid.Parse(params.ByName("reagentID"))
	if err != nil {
		rc.logger.Info("Not found")
		common.ErrorResp(w, common.NotFound)
		return
	}
	errs := db.PerformBatch(r.Context(), rc.dbpool, []db.BatchSet{reagent.Update})
	reagentErr := errs[0]
	if reagentErr != nil {
		errStruct := db.ErrorAsStruct(reagentErr)
		switch errStruct.(type) {
		case db.UniqueViolation:
			err = errStruct.(db.UniqueViolation).LocalizeUniqueViolation(db.Reagent{})
			rc.logger.Info(err.Error())
			errMap := err.(db.DBError).Map()
			reagentPostErrMapAddInput(errMap, reagent, rc.userID)
			errTmpl.Execute(w, errMap)
			return
		default:
			rc.logger.Error(reagentErr.Error())
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
	reagent.ID, err = uuid.Parse(params.ByName("reagentID"))
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
	reagentID, err := uuid.Parse(params.ByName("reagentID"))
	if err != nil {
		rc.logger.Info("Invalid UUID")
		common.ErrorResp(w, common.NotFound)
		return
	}
	reagent := db.Reagent{ID: reagentID}
	errs := db.PerformBatch(r.Context(), rc.dbpool, []db.BatchSet{reagent.Get})
	reagentErr := errs[0]
	if reagentErr != nil {
		errStruct := db.ErrorAsStruct(reagentErr)
		switch errStruct.(type) {
		case db.DoesNotExist:
			rc.logger.Info("Not found")
			common.ErrorResp(w, common.NotFound)
			return
		default:
			rc.logger.Error(reagentErr.Error())
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
		ID:      reagent.ID.String(),
		Name:    reagent.Name,
		Formula: reagent.Formula,
	}
	tmpl.Execute(w, data)
}
