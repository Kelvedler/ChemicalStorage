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
	"github.com/Kelvedler/ChemicalStorage/pkg/middleware"
)

type reagentsData struct {
	ReagentsSlice []db.Reagent
	LastReagent   db.Reagent
	NextOffset    int
	Src           string
	Caller        db.StorageUser
}

func (data *reagentsData) set(reagentsSlice []db.Reagent, src string, limit, offset int) {
	if len(reagentsSlice) >= limit {
		data.ReagentsSlice = reagentsSlice[:len(reagentsSlice)-1]
		data.LastReagent = reagentsSlice[len(reagentsSlice)-1]
		data.NextOffset = offset + len(reagentsSlice)
		data.Src = src
	} else {
		data.ReagentsSlice = reagentsSlice
	}
}

type reagentData struct {
	Caller             db.StorageUser
	ID                 string
	Name               string
	Formula            string
	NameErr            string
	FormulaErr         string
	PostXsrf           string
	PutXsrf            string
	InstancesSlice     []db.ReagentInstanceExtended
	UsedInstancesSlice []db.ReagentInstanceExtended
}

func (data *reagentData) addInstances(instancesSlice []db.ReagentInstanceExtended) {
	for _, inst := range instancesSlice {
		if inst.ReagentInstance.UsedAt.IsZero() {
			data.InstancesSlice = append(data.InstancesSlice, inst)
		} else {
			data.UsedInstancesSlice = append(data.UsedInstancesSlice, inst)
		}
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
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	src := ""
	limit := 24
	offset := 0
	reagentsRange := db.ReagentsRange{
		Limit:  limit,
		Offset: offset,
		Src:    src,
	}
	caller := db.StorageUser{ID: rc.UserID}
	errs := db.PerformBatch(
		r.Context(),
		rc.DBpool,
		[]db.BatchSet{reagentsRange.Get, caller.GetByID},
	)
	reagentsErr := errs[0]
	if reagentsErr != nil {
		rc.Logger.Error(reagentsErr.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	data := reagentsData{Caller: caller}
	data.set(reagentsRange.Reagents, src, limit, offset)
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
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	reagentID, err := uuid.Parse(params.ByName("reagentID"))
	if err != nil {
		rc.Logger.Info("Invalid UUID")
		common.ErrorResp(w, common.NotFound)
		return
	}

	reagent := db.Reagent{ID: reagentID}
	rir := db.ReagentInstanceRange{
		ReagentID: reagentID,
	}
	caller := db.StorageUser{ID: rc.UserID}
	errs := db.PerformBatch(
		r.Context(),
		rc.DBpool,
		[]db.BatchSet{reagent.Get, rir.Get, caller.GetByID},
	)
	reagentErr := errs[0]
	reagentInstanceErr := errs[1]
	if reagentErr != nil {
		errStruct := db.ErrorAsStruct(reagentErr)
		switch errStruct.(type) {
		case db.DoesNotExist:
			rc.Logger.Info("Not found")
			common.ErrorResp(w, common.NotFound)
			return
		default:
			rc.Logger.Error(reagentErr.Error())
			common.ErrorResp(w, common.Internal)
			return
		}
	}
	if reagentInstanceErr != nil {
		rc.Logger.Error(reagentInstanceErr.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	data := reagentData{
		Caller:  caller,
		ID:      reagent.ID.String(),
		Name:    reagent.Name,
		Formula: reagent.Formula,
		PutXsrf: getReagentPutXsrf(rc.UserID, reagentID),
	}
	data.addInstances(rir.ReagentInstancesExtended)
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
	rc *middleware.RequestContext,
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
	caller := db.StorageUser{ID: rc.UserID}
	_ = db.PerformBatch(r.Context(), rc.DBpool, []db.BatchSet{caller.GetByID})
	data := reagentData{
		Caller:   caller,
		PostXsrf: getReagentPostXsrf(rc.UserID),
	}
	tmpl.Execute(w, data)
}

func ReagentsAPI(
	rc *middleware.RequestContext,
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
		rc.Logger.Info(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	target := r.URL.Query().Get("target")

	searchForm := SearchAPIForm{Src: src, Offset: offset, Target: target}
	err = rc.Validate.Struct(searchForm)
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), searchForm)
		rc.Logger.Info(err.Error())
		w.WriteHeader(400)
		return
	}
	limit := 24
	reagentsRange := db.ReagentsRange{
		Limit:  limit,
		Offset: offset,
		Src:    searchForm.Src,
	}
	errs := db.PerformBatch(r.Context(), rc.DBpool, []db.BatchSet{reagentsRange.Get})
	reagentsErr := errs[0]
	if reagentsErr != nil {
		rc.Logger.Error(reagentsErr.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	var data reagentsData
	data.set(reagentsRange.Reagents, src, limit, offset)
	data.Caller = db.StorageUser{ID: rc.UserID, Role: rc.UserRole}
	tmpl := template.Must(
		template.ParseFiles(
			"templates/reagents-assets.html",
		),
	)
	if searchForm.Target == "grid" {
		tmpl = tmpl.Lookup("reagents-grid")
	} else {
		tmpl = tmpl.Lookup("reagents-search")
	}
	tmpl.Execute(w, data)
}

func sanitizeReagent(rc *middleware.RequestContext, reagent *db.Reagent) {
	sanitizer := rc.Sanitize
	reagent.Name = sanitizer.Sanitize(reagent.Name)
	reagent.Formula = sanitizer.Sanitize(reagent.Formula)
}

func ReagentCreateAPI(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	var reagent db.Reagent
	err := common.BindJSON(r, &reagent)
	if err != nil {
		rc.Logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}

	sanitizeReagent(rc, &reagent)
	tmpl := template.Must(template.ParseFiles("templates/reagents-assets.html")).
		Lookup("reagent-form")

	err = rc.Validate.StructPartial(reagent, "Name", "Formula")
	var data reagentData
	data.Caller.ID = rc.UserID
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), reagent)
		rc.Logger.Info(err.Error())
		errMap := err.(common.ValidationError).Map()
		data.NameErr = errMap["NameErr"]
		data.FormulaErr = errMap["FormulaErr"]
		tmpl.Execute(w, data)
		return
	}
	errs := db.PerformBatch(r.Context(), rc.DBpool, []db.BatchSet{reagent.Create})
	reagentErr := errs[0]
	if reagentErr != nil {
		errStruct := db.ErrorAsStruct(reagentErr)
		switch errStruct.(type) {
		case db.UniqueViolation:
			err = errStruct.(db.UniqueViolation).Localize(db.Reagent{})
			rc.Logger.Info(err.Error())
			data.FormulaErr = err.(db.DBError).Map()["FormulaErr"]
			tmpl.Execute(w, data)
			return
		default:
			rc.Logger.Error(reagentErr.Error())
			common.ErrorResp(w, common.Internal)
			return
		}
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/reagents/%s", reagent.ID))
}

func ReagentPutAPI(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	var reagent db.Reagent
	err := common.BindJSON(r, &reagent)
	if err != nil {
		rc.Logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	sanitizeReagent(rc, &reagent)
	tmpl := template.Must(template.ParseFiles("templates/reagents-assets.html"))

	errTmpl := tmpl.Lookup("reagent-form")

	err = rc.Validate.StructPartial(reagent, "Name", "Formula")
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), reagent)
		rc.Logger.Info(err.Error())
		errMap := err.(common.ValidationError).Map()
		w.Header().Set("HX-Retarget", "#reagent-form")
		errTmpl.Execute(w, errMap)
		return
	}
	reagent.ID, err = uuid.Parse(params.ByName("reagentID"))
	if err != nil {
		rc.Logger.Info("Not found")
		common.ErrorResp(w, common.NotFound)
		return
	}
	errs := db.PerformBatch(r.Context(), rc.DBpool, []db.BatchSet{reagent.Update})
	reagentErr := errs[0]
	if reagentErr != nil {
		errStruct := db.ErrorAsStruct(reagentErr)
		switch errStruct.(type) {
		case db.UniqueViolation:
			err = errStruct.(db.UniqueViolation).Localize(db.Reagent{})
			rc.Logger.Info(err.Error())
			errMap := err.(db.DBError).Map()
			w.Header().Set("HX-Retarget", "#reagent-form")
			errTmpl.Execute(w, errMap)
			return
		default:
			rc.Logger.Error(reagentErr.Error())
			common.ErrorResp(w, common.Internal)
			return
		}
	}
	successTmpl := tmpl.Lookup("reagent")
	caller := db.StorageUser{
		Role: rc.UserRole,
	}
	data := reagentData{
		Caller:  caller,
		ID:      reagent.ID.String(),
		Name:    reagent.Name,
		Formula: reagent.Formula,
		PutXsrf: getReagentPutXsrf(rc.UserID, reagent.ID),
	}
	successTmpl.Execute(w, data)
}
