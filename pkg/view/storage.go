package view

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/xsrftoken"

	"github.com/Kelvedler/ChemicalStorage/pkg/common"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
	"github.com/Kelvedler/ChemicalStorage/pkg/env"
	"github.com/Kelvedler/ChemicalStorage/pkg/middleware"
)

type storagesData struct {
	Caller        db.StorageUser
	StoragesSlice []db.Storage
	LastStorage   db.Storage
	NextOffset    int
	Src           string
}

type storageData struct {
	Caller   db.StorageUser
	ID       string
	Name     string
	NameErr  string
	Cells    int
	CellsErr string
	PostXsrf string
}

func (s *storagesData) set(storagesSlice []db.Storage, src string, offset int) {
	if len(storagesSlice) > 1 {
		s.StoragesSlice = storagesSlice[:len(storagesSlice)-1]
		s.LastStorage = storagesSlice[len(storagesSlice)-1]
		s.NextOffset = offset + len(storagesSlice)
		s.Src = src
	} else {
		s.StoragesSlice = storagesSlice
	}
}

func Storages(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	offset := 0
	src := ""
	storagesRange := db.StoragesRange{
		Limit:  20,
		Offset: offset,
		Src:    src,
	}
	caller := db.StorageUser{ID: rc.UserID}
	errs := db.PerformBatch(
		r.Context(),
		rc.DBpool,
		[]db.BatchSet{storagesRange.Get, caller.GetByID},
	)
	storagesErr := errs[0]
	if storagesErr != nil {
		rc.Logger.Error(storagesErr.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	data := storagesData{Caller: caller}
	data.set(storagesRange.Storages, src, offset)
	tmpl := template.Must(
		template.ParseFiles(
			"templates/storages.html",
			"templates/storages-assets.html",
			"templates/base.html",
		),
	)
	tmpl.Execute(w, data)
}

func StoragesAPI(
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
		return
	}
	searchForm := SearchAPIForm{Src: src, Offset: offset}
	err = rc.Validate.Struct(searchForm)
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), searchForm)
		rc.Logger.Info(err.Error())
		w.WriteHeader(400)
		return
	}
	storagesRange := db.StoragesRange{
		Limit:  20,
		Offset: offset,
		Src:    src,
	}
	errs := db.PerformBatch(r.Context(), rc.DBpool, []db.BatchSet{storagesRange.Get})
	storagesErr := errs[0]
	if storagesErr != nil {
		rc.Logger.Error(storagesErr.Error())
		w.WriteHeader(400)
		return
	}
	var data storagesData
	data.set(storagesRange.Storages, src, offset)
	tmpl := template.Must(template.ParseFiles("templates/storages-assets.html")).
		Lookup("storages-search")
	tmpl.Execute(w, data)
}

func Storage(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	storageID, err := uuid.Parse(params.ByName("storageID"))
	if err != nil {
		rc.Logger.Info("Invalid UUID")
		common.ErrorResp(w, common.NotFound)
		return
	}
	storage := db.Storage{ID: storageID}
	caller := db.StorageUser{ID: rc.UserID}
	errs := db.PerformBatch(r.Context(), rc.DBpool, []db.BatchSet{storage.Get, caller.GetByID})
	storageErr := errs[0]
	if storageErr != nil {
		errStruct := db.ErrorAsStruct(storageErr)
		switch errStruct.(type) {
		case db.DoesNotExist:
			rc.Logger.Info("Not found")
			common.ErrorResp(w, common.NotFound)
			return
		default:
			rc.Logger.Error(storageErr.Error())
			common.ErrorResp(w, common.Internal)
			return
		}
	}
	data := storageData{
		Caller: caller,
		ID:     storage.ID.String(),
		Name:   storage.Name,
	}
	tmpl := template.Must(template.ParseFiles("templates/storage.html", "templates/base.html"))
	tmpl.Execute(w, data)
}

func getStoragePostXsrf(userID uuid.UUID) string {
	return xsrftoken.Generate(
		env.Env.SecretKey,
		userID.String(),
		"/api/v1/storages",
	)
}

func sanitizeStorage(rc *middleware.RequestContext, storage *db.Storage) {
	sanitizer := rc.Sanitize
	storage.Name = sanitizer.Sanitize(storage.Name)
}

func storageErrMapAddInput(errMap map[string]string, storage db.Storage) {
	errMap["Name"] = storage.Name
	errMap["Cells"] = strconv.Itoa(int(storage.Cells))
}

func storagePostErrMapAddInput(
	errMap map[string]string,
	storage db.Storage,
	userID uuid.UUID,
) {
	storageErrMapAddInput(errMap, storage)
	errMap["PostXsrf"] = getStoragePostXsrf(userID)
}

func StorageCreate(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	tmpl := template.Must(
		template.ParseFiles(
			"templates/storage-new.html",
			"templates/base.html",
			"templates/storages-assets.html",
		),
	)
	caller := db.StorageUser{ID: rc.UserID}
	_ = db.PerformBatch(r.Context(), rc.DBpool, []db.BatchSet{caller.GetByID})
	data := storageData{
		Caller:   caller,
		PostXsrf: getStoragePostXsrf(rc.UserID),
	}
	tmpl.Execute(w, data)
}

func StorageCreateAPI(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	var input db.StorageInput
	err := common.BindJSON(r, &input)
	if err != nil {
		rc.Logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	storage, err := input.Bind()
	if err != nil {
		rc.Logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	sanitizeStorage(rc, &storage)
	tmpl := template.Must(template.ParseFiles("templates/storages-assets.html")).
		Lookup("storage-form")
	err = rc.Validate.StructPartial(storage, "Name", "Cells")
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), storage)
		rc.Logger.Info(err.Error())
		errMap := err.(common.ValidationError).Map()
		storagePostErrMapAddInput(errMap, storage, rc.UserID)
		tmpl.Execute(w, errMap)
		return
	}
	errs := db.PerformBatch(r.Context(), rc.DBpool, []db.BatchSet{storage.Create})
	storageErr := errs[0]
	if storageErr != nil {
		rc.Logger.Error(storageErr.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	w.Header().Set("HX-Redirect", fmt.Sprintf("/storages/%s", storage.ID))
}
