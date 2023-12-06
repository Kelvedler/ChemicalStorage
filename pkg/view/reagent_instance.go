package view

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/xsrftoken"

	"github.com/Kelvedler/ChemicalStorage/pkg/common"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
	"github.com/Kelvedler/ChemicalStorage/pkg/env"
	"github.com/Kelvedler/ChemicalStorage/pkg/middleware"
)

type instanceData struct {
	Caller         db.StorageUser
	ID             uuid.UUID
	Reagent        db.Reagent
	Storage        db.Storage
	StorageCell    db.StorageCell
	UsedAt         time.Time
	DeletedAt      time.Time
	ExpiresAt      time.Time
	Err            string
	ExpiresAtErr   string
	CellErr        string
	StoragesSlice  []db.Storage
	CreateXsrf     string
	UseXsrf        string
	TransferXsrf   string
	EditState      bool
	ReloadData     bool
	ReloadUsedAt   bool
	ReloadStorages bool
}

func getInstanceCreateXsrf(userID, reagentID uuid.UUID) string {
	return xsrftoken.Generate(
		env.Env.SecretKey,
		userID.String(),
		fmt.Sprintf("/api/v1/reagents/%s/instances", reagentID),
	)
}

func getInstanceUseXsrf(userID, instanceID, reagentID uuid.UUID) string {
	return xsrftoken.Generate(
		env.Env.SecretKey,
		userID.String(),
		fmt.Sprintf("/api/v1/reagents/%s/instances/%s/use", reagentID, instanceID),
	)
}

func getInstanceTranserXsrf(userID, instanceID, reagentID uuid.UUID) string {
	return xsrftoken.Generate(
		env.Env.SecretKey,
		userID.String(),
		fmt.Sprintf("/api/v1/reagents/%s/instances/%s/transfer", reagentID, instanceID),
	)
}

func ReagentInstanceCreate(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	reagentID, err := uuid.Parse(params.ByName("reagentID"))
	if err != nil {
		rc.Logger.Info(err.Error())
		common.ErrorResp(w, common.NotFound)
		return
	}
	tmpl := template.Must(
		template.ParseFiles(
			"templates/instance-new.html",
			"templates/instances-assets.html",
			"templates/storages-assets.html",
			"templates/base.html",
		),
	)
	storagesRange := db.StoragesRange{
		Limit:  40,
		Offset: 0,
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
	data := instanceData{
		Caller:        caller,
		Reagent:       db.Reagent{ID: reagentID},
		StoragesSlice: storagesRange.Storages,
		CreateXsrf:    getInstanceCreateXsrf(caller.ID, reagentID),
	}
	tmpl.Execute(w, data)
}

func sanitizeReagentInstance(rc *middleware.RequestContext, input *reagentInstanceInput) {
	sanitizer := rc.Sanitize
	input.ExpiresAt = sanitizer.Sanitize(input.ExpiresAt)
	input.Storage = sanitizer.Sanitize(input.Storage)
	input.Cell = sanitizer.Sanitize(input.Cell)
}

type reagentInstanceInput struct {
	ExpiresAt string `json:"expires_at"`
	Storage   string `json:"storage"`
	Cell      string `json:"cell"`
}

type reagentInstance struct {
	ExpiresAt time.Time `json:"expires_at" validate:"gt" uaLocal:"термін придатності"`
	Storage   uuid.UUID `json:"storage"`
	Cell      int16     `json:"cell"                     uaLocal:"відділ"`
}

func (input reagentInstanceInput) Bind() (output reagentInstance, err error) {
	if input.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.DateOnly, input.ExpiresAt)
		if err != nil {
			return reagentInstance{}, err
		}
		output.ExpiresAt = expiresAt
	}
	if input.Storage != "" {
		storage, err := uuid.Parse(input.Storage)
		if err != nil {
			return reagentInstance{}, err
		}
		output.Storage = storage
	}
	if input.Cell != "" {
		cell, err := strconv.Atoi(input.Cell)
		if err != nil {
			return reagentInstance{}, err
		}
		output.Cell = int16(cell)
	}
	return output, nil
}

func ReagentInstanceCreateAPI(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	var inputStr reagentInstanceInput
	err := common.BindJSON(r, &inputStr)
	if err != nil {
		rc.Logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	sanitizeReagentInstance(rc, &inputStr)
	input, err := inputStr.Bind()
	if err != nil {
		rc.Logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	tmpl := template.Must(template.ParseFiles("templates/instances-assets.html", "templates/storages-assets.html")).
		Lookup("instance-form")
	returnData := instanceData{ReloadData: true}
	err = rc.Validate.Struct(input)
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), input)
		rc.Logger.Info(err.Error())
		returnData.ExpiresAtErr = err.(common.ValidationError).Map()["ExpiresAtErr"]
		tmpl.Execute(w, returnData)
		return
	}
	reagentID, err := uuid.Parse(params.ByName("reagentID"))
	if err != nil {
		rc.Logger.Info(err.Error())
		common.ErrorResp(w, common.NotFound)
		return
	}
	reagentInstance := db.ReagentInstance{
		Reagent:   reagentID,
		ExpiresAt: input.ExpiresAt,
	}
	storageCell := db.StorageCell{
		Storage: input.Storage,
		Number:  input.Cell,
	}
	reagentInstanceExtended := db.ReagentInstanceExtended{
		ReagentInstance: reagentInstance,
		Storage:         db.Storage{ID: input.Storage},
		StorageCell:     storageCell,
	}
	errs := db.PerformBatch(r.Context(), rc.DBpool, []db.BatchSet{
		storageCell.TryCreate,
		reagentInstanceExtended.Create,
	})
	for _, err = range errs {
		if err != nil {
			errStruct := db.ErrorAsStruct(err)
			switch errStruct.(type) {
			case db.OutOfLimits:
				err = errStruct.(db.OutOfLimits).Localize(storageCell)
				rc.Logger.Info(err.Error())
				returnData.CellErr = err.(db.DBError).Map()["NumberErr"]
				tmpl.Execute(w, returnData)
			default:
				rc.Logger.Error(err.Error())
				common.ErrorResp(w, common.Internal)
			}
			return
		}
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/reagents/%s", reagentInstance.Reagent))
}

func ReagentInstance(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	reagentID, reagentErr := uuid.Parse(params.ByName("reagentID"))
	instanceID, instanceErr := uuid.Parse(params.ByName("instanceID"))
	for _, err := range []error{reagentErr, instanceErr} {
		if err != nil {
			rc.Logger.Info(err.Error())
			common.ErrorResp(w, common.NotFound)
			return
		}
	}
	caller := db.StorageUser{ID: rc.UserID}
	rie := db.ReagentInstanceExtended{
		ReagentInstance: db.ReagentInstance{ID: instanceID, Reagent: reagentID},
	}
	storagesRange := db.StoragesRange{
		Limit:  40,
		Offset: 0,
	}
	errs := db.PerformBatch(
		r.Context(),
		rc.DBpool,
		[]db.BatchSet{rie.Get, storagesRange.Get, caller.GetByID},
	)
	for i, err := range errs {
		if err != nil {
			errStruct := db.ErrorAsStruct(err)
			switch errStruct.(type) {
			case db.DoesNotExist:
				if i == 2 {
					rc.Logger.Info("Unauthorized")
					common.ErrorResp(w, common.Unauthorized)
				} else {
					rc.Logger.Info("Not found")
					common.ErrorResp(w, common.NotFound)
				}
			default:
				rc.Logger.Error(instanceErr.Error())
				common.ErrorResp(w, common.Internal)
			}
			return
		}
	}

	data := instanceData{
		Caller:        caller,
		ID:            rie.ReagentInstance.ID,
		UsedAt:        rie.ReagentInstance.UsedAt,
		ExpiresAt:     rie.ReagentInstance.ExpiresAt,
		Reagent:       rie.Reagent,
		Storage:       rie.Storage,
		StorageCell:   rie.StorageCell,
		StoragesSlice: storagesRange.Storages,
		UseXsrf:       getInstanceUseXsrf(rc.UserID, rie.ReagentInstance.ID, rie.Reagent.ID),
		TransferXsrf:  getInstanceTranserXsrf(rc.UserID, rie.ReagentInstance.ID, rie.Reagent.ID),
	}
	tmpl := template.Must(
		template.ParseFiles(
			"templates/instance.html",
			"templates/base.html",
			"templates/instances-assets.html",
			"templates/storages-assets.html",
		),
	)
	tmpl.Execute(w, data)
}

func ReagentInstanceUseAPI(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	reagentID, reagentErr := uuid.Parse(params.ByName("reagentID"))
	instanceID, instanceErr := uuid.Parse(params.ByName("instanceID"))
	for _, err := range []error{reagentErr, instanceErr} {
		if err != nil {
			rc.Logger.Info(err.Error())
			common.ErrorResp(w, common.NotFound)
			return
		}
	}
	tmpl := template.Must(template.ParseFiles("templates/instances-assets.html", "templates/storages-assets.html")).
		Lookup("instance")

	rie := db.ReagentInstanceExtended{
		ReagentInstance: db.ReagentInstance{ID: instanceID, Reagent: reagentID, UsedAt: time.Now()},
	}
	errs := db.PerformBatch(r.Context(), rc.DBpool, []db.BatchSet{rie.Update})
	instanceErr = errs[0]
	if instanceErr != nil {
		errStruct := db.ErrorAsStruct(instanceErr)
		switch errStruct.(type) {
		case db.AlreadySet:
			rc.Logger.Info(instanceErr.Error())
			common.ErrorResp(w, common.Internal)
		case db.DoesNotExist:
			rc.Logger.Info(instanceErr.Error())
			common.ErrorResp(w, common.NotFound)
		default:
			rc.Logger.Error(instanceErr.Error())
			common.ErrorResp(w, common.Internal)
		}
		return
	}
	data := instanceData{
		Caller:       db.StorageUser{ID: rc.UserID, Role: rc.UserRole},
		UsedAt:       rie.ReagentInstance.UsedAt,
		ReloadUsedAt: true,
	}
	tmpl.Execute(w, data)
}

func ReagentInstanceTransferAPI(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	reagentID, reagentErr := uuid.Parse(params.ByName("reagentID"))
	instanceID, instanceErr := uuid.Parse(params.ByName("instanceID"))
	for _, err := range []error{reagentErr, instanceErr} {
		if err != nil {
			rc.Logger.Info(err.Error())
			common.ErrorResp(w, common.NotFound)
			return
		}
	}
	var inputStr reagentInstanceInput
	err := common.BindJSON(r, &inputStr)
	if err != nil {
		rc.Logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	sanitizeReagentInstance(rc, &inputStr)
	input, err := inputStr.Bind()
	if err != nil {
		rc.Logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	storageCell := db.StorageCell{
		Storage: input.Storage,
		Number:  input.Cell,
	}
	rie := db.ReagentInstanceExtended{
		ReagentInstance: db.ReagentInstance{ID: instanceID, Reagent: reagentID},
		StorageCell:     storageCell,
		Storage:         db.Storage{ID: input.Storage},
	}
	errs := db.PerformBatch(
		r.Context(),
		rc.DBpool,
		[]db.BatchSet{storageCell.TryCreate, rie.Update},
	)
	tmpl := template.Must(
		template.ParseFiles("templates/instances-assets.html", "templates/storages-assets.html"),
	).Lookup("instance")
	data := instanceData{
		Caller:         db.StorageUser{ID: rc.UserID, Role: rc.UserRole},
		StorageCell:    rie.StorageCell,
		ReloadStorages: true,
	}
	for _, err = range errs {
		if err != nil {
			errStruct := db.ErrorAsStruct(err)
			switch errStruct.(type) {
			case db.OutOfLimits:
				err = errStruct.(db.OutOfLimits).Localize(storageCell)
				rc.Logger.Info(err.Error())
				data.CellErr = err.(db.DBError).Map()["NumberErr"]
				data.EditState = true
				tmpl.Execute(w, data)
			default:
				rc.Logger.Error(err.Error())
				common.ErrorResp(w, common.Internal)
			}
			return
		}
	}
	w.Header().Set("HX-Redirect", fmt.Sprintf("/reagents/%s/instances/%s", reagentID, instanceID))
	tmpl.Execute(w, nil)
}
