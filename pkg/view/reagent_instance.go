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
)

type instanceData struct {
	Caller        db.StorageUser
	ReagentID     string
	ExpiresAtErr  string
	CellErr       string
	StoragesSlice []db.Storage
	PostXsrf      string
}

func getInstancePostXsrf(userID, reagentID string) string {
	return xsrftoken.Generate(
		env.Env.SecretKey,
		userID,
		fmt.Sprintf("/api/v1/reagents/%s/instances", reagentID),
	)
}

func ReagentInstanceCreate(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	reagentID := params.ByName("reagentID")
	tmpl := template.Must(
		template.ParseFiles(
			"templates/instance-new.html",
			"templates/instances-assets.html",
			"templates/base.html",
		),
	)
	storagesRange := db.StoragesRange{
		Limit:  40,
		Offset: 0,
	}
	caller := db.StorageUser{ID: rc.userID}
	errs := db.PerformBatch(
		r.Context(),
		rc.dbpool,
		[]db.BatchSet{storagesRange.Get, caller.GetByID},
	)
	storagesErr := errs[0]
	if storagesErr != nil {
		rc.logger.Error(storagesErr.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	data := instanceData{
		Caller:        caller,
		ReagentID:     reagentID,
		StoragesSlice: storagesRange.Storages,
		PostXsrf:      getInstancePostXsrf(caller.ID.String(), reagentID),
	}
	tmpl.Execute(w, data)
}

func sanitizeReagentInstance(rc *RequestContext, input *reagentInstanceCreateInput) {
	sanitizer := rc.sanitize
	input.ExpiresAt = sanitizer.Sanitize(input.ExpiresAt)
	input.Storage = sanitizer.Sanitize(input.Storage)
	input.Cell = sanitizer.Sanitize(input.Cell)
}

type reagentInstanceCreateInput struct {
	ExpiresAt string `json:"expires_at"`
	Storage   string `json:"storage"`
	Cell      string `json:"cell"`
}

type reagentInstanceCreate struct {
	ExpiresAt time.Time `json:"expires_at" validate:"gt"             uaLocal:"термін придатності"`
	Storage   uuid.UUID `json:"storage"`
	Cell      int16     `json:"cell"       validate:"gte=1,lte=1000" uaLocal:"відділ"`
}

func (input reagentInstanceCreateInput) Bind() (output reagentInstanceCreate, err error) {
	if input.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.DateOnly, input.ExpiresAt)
		if err != nil {
			return reagentInstanceCreate{}, err
		}
		output.ExpiresAt = expiresAt
	}
	if input.Storage != "" {
		storage, err := uuid.Parse(input.Storage)
		if err != nil {
			return reagentInstanceCreate{}, err
		}
		output.Storage = storage
	}
	if input.Cell != "" {
		cell, err := strconv.Atoi(input.Cell)
		if err != nil {
			return reagentInstanceCreate{}, err
		}
		output.Cell = int16(cell)
	}
	return output, nil
}

func ReagentInstanceCreateAPI(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	var inputStr reagentInstanceCreateInput
	err := common.BindJSON(r, &inputStr)
	if err != nil {
		rc.logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	sanitizeReagentInstance(rc, &inputStr)
	input, err := inputStr.Bind()
	tmpl := template.Must(template.ParseFiles("templates/instances-assets.html")).
		Lookup("instance-form")
	err = rc.validate.Struct(input)
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), input)
		rc.logger.Info(err.Error())
		errMap := err.(common.ValidationError).Map()
		tmpl.Execute(w, errMap)
		return
	}
	reagentID, err := uuid.Parse(params.ByName("reagentID"))
	if err != nil {
		rc.logger.Info(err.Error())
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
		Storage:         input.Storage,
		CellNumber:      input.Cell,
	}
	errs := db.PerformBatch(r.Context(), rc.dbpool, []db.BatchSet{
		storageCell.TryCreate,
		reagentInstanceExtended.Create,
	})
	for _, err = range errs {
		if err != nil {
			rc.logger.Error(err.Error())
			common.ErrorResp(w, common.Internal)
			return
		}
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/reagents/%s", reagentInstance.Reagent))
}
