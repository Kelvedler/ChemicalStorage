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

type storageUsersData struct {
	StorageUsersSlice []db.StorageUser
	LastStorageUser   db.StorageUser
	NextOffset        int
	Src               string
	Caller            db.StorageUser
}

func (s *storageUsersData) set(storageUsersSlice []db.StorageUser, src string, offset int) {
	if len(storageUsersSlice) > 1 {
		s.StorageUsersSlice = storageUsersSlice[:len(storageUsersSlice)-1]
		s.LastStorageUser = storageUsersSlice[len(storageUsersSlice)-1]
		s.NextOffset = offset + len(storageUsersSlice)
		s.Src = src
	} else {
		s.StorageUsersSlice = storageUsersSlice
	}
}

func Users(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	offset := 0
	src := ""
	storageUsersRange := db.StorageUsersRange{
		Limit:     20,
		Offset:    offset,
		Src:       src,
		ExcludeID: rc.UserID,
	}
	caller := db.StorageUser{ID: rc.UserID}
	errs := db.PerformBatch(
		r.Context(),
		rc.DBpool,
		[]db.BatchSet{storageUsersRange.Get, caller.GetByID},
	)
	storageUsersErr := errs[0]
	if storageUsersErr != nil {
		rc.Logger.Error(storageUsersErr.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	data := storageUsersData{Caller: caller}
	data.set(storageUsersRange.StorageUsers, src, offset)
	tmpl := template.Must(
		template.ParseFiles(
			"templates/users.html",
			"templates/base.html",
			"templates/users-assets.html",
		),
	)
	tmpl.Execute(w, data)
}

type userByIDData struct {
	User        db.StorageUser
	Caller      db.StorageUser
	UserPutXsrf string
}

func User(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	userID, err := uuid.Parse(params.ByName("userID"))
	if err != nil {
		rc.Logger.Info("Invalid UUID")
		common.ErrorResp(w, common.NotFound)
	}
	if userID == rc.UserID {
		rc.Logger.Info("Cannot view self")
		common.ErrorResp(w, common.Forbidden)
		return
	}
	storageUser := db.StorageUser{ID: userID}
	caller := db.StorageUser{ID: rc.UserID}
	errs := db.PerformBatch(
		r.Context(),
		rc.DBpool,
		[]db.BatchSet{storageUser.GetByID, caller.GetByID},
	)
	storageUserErr := errs[0]
	if storageUserErr != nil {
		errStruct := db.ErrorAsStruct(storageUserErr)
		switch errStruct.(type) {
		case db.InvalidUUID, db.DoesNotExist:
			rc.Logger.Info("Not found")
			common.ErrorResp(w, common.NotFound)
			return
		default:
			rc.Logger.Error(storageUserErr.Error())
			common.ErrorResp(w, common.Internal)
			return
		}
	}
	userPutXsrf := xsrftoken.Generate(
		env.Env.SecretKey,
		caller.ID.String(),
		fmt.Sprintf("/api/v1/users/%s", userID),
	)
	data := userByIDData{
		Caller:      caller,
		User:        storageUser,
		UserPutXsrf: userPutXsrf,
	}
	tmpl := template.Must(
		template.ParseFiles(
			"templates/user.html",
			"templates/base.html",
		),
	)
	tmpl.Execute(w, data)
}

type UsersAPIForm struct {
	Src    string `json:"src"    validate:"omitempty,lte=50"          uaLocal:"пошук"`
	Offset int    `json:"offset" validate:"omitempty,min=0,max=10000" uaLocal:"зміщення"`
}

func UsersAPI(
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
	srcForm := UsersAPIForm{Src: src, Offset: offset}
	err = rc.Validate.Struct(srcForm)
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), srcForm)
		rc.Logger.Info(err.Error())
		return
	}
	storageUsersRange := db.StorageUsersRange{
		Limit:     20,
		Offset:    offset,
		Src:       srcForm.Src,
		ExcludeID: rc.UserID,
	}
	caller := db.StorageUser{ID: rc.UserID}
	errs := db.PerformBatch(
		r.Context(),
		rc.DBpool,
		[]db.BatchSet{storageUsersRange.Get, caller.GetByID},
	)
	if errs[0] != nil {
		rc.Logger.Error(errs[0].Error())
		return
	}
	data := storageUsersData{Caller: caller}
	data.set(storageUsersRange.StorageUsers, src, offset)
	tmpl := template.Must(
		template.ParseFiles(
			"templates/users.html",
			"templates/users-assets.html",
		),
	).Lookup("users-search")
	tmpl.Execute(w, data)
}

func UserPutAPI(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	var userInput db.StorageUserInput
	err := common.BindJSON(r, &userInput)
	if err != nil {
		rc.Logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	userInput.ID = params.ByName("userID")
	if userInput.ID == rc.UserID.String() {
		rc.Logger.Info("Cannot update self")
		common.ErrorResp(w, common.Forbidden)
		return
	}
	user, err := userInput.Bind()
	if err != nil {
		rc.Logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	if user.Role == db.Admin {
		rc.Logger.Info("Can't set admin role")
		w.WriteHeader(403)
		return
	}
	err = rc.Validate.StructPartial(user, "Active")
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), user)
		rc.Logger.Info(err.Error())
		return
	}
	errs := db.PerformBatch(r.Context(), rc.DBpool, []db.BatchSet{user.Update})
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
}
