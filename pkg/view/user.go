package view

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/xsrftoken"

	"github.com/Kelvedler/ChemicalStorage/pkg/common"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
	"github.com/Kelvedler/ChemicalStorage/pkg/env"
)

type storageUsersData struct {
	StorageUsersSlice []db.StorageUser
	LastStorageUser   db.StorageUser
	NextOffset        int
	Src               string
	Caller            db.StorageUser
}

func newStorageUsersData(
	storageUsersSlice []db.StorageUser,
	src string,
	offset int,
) (data storageUsersData) {
	if len(storageUsersSlice) > 1 {
		data.StorageUsersSlice = storageUsersSlice[:len(storageUsersSlice)-1]
		data.LastStorageUser = storageUsersSlice[len(storageUsersSlice)-1]
		data.NextOffset = offset + len(storageUsersSlice)
		data.Src = src
	} else {
		data.StorageUsersSlice = storageUsersSlice
	}
	return data
}

func Users(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	offset := 0
	src := ""
	storageUsersSlice, err := db.StorageUserGetRange(
		r.Context(),
		rc.dbpool,
		20,
		offset,
		src,
		rc.userID,
	)
	if err != nil {
		rc.logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	data := newStorageUsersData(storageUsersSlice, src, offset)
	storageUser, _ := db.StorageUserGetByID(r.Context(), rc.dbpool, rc.userID)
	data.Caller = storageUser
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
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	user_id := params.ByName("id")
	if user_id == rc.userID {
		rc.logger.Info("Cannot view self")
		common.ErrorResp(w, common.Forbidden)
		return
	}
	storageUser, err := db.StorageUserGetByID(r.Context(), rc.dbpool, user_id)
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
	caller, _ := db.StorageUserGetByID(r.Context(), rc.dbpool, rc.userID)
	userPutXsrf := xsrftoken.Generate(
		env.Env.SecretKey,
		caller.ID.String(),
		fmt.Sprintf("/api/v1/users/%s", user_id),
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
	srcForm := UsersAPIForm{Src: src, Offset: offset}
	err = rc.validate.Struct(srcForm)
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), srcForm)
		rc.logger.Info(err.Error())
		w.WriteHeader(400)
		return
	}
	storageUsersSlice, err := db.StorageUserGetRange(
		r.Context(),
		rc.dbpool,
		20,
		offset,
		srcForm.Src,
		rc.userID,
	)
	if err != nil {
		rc.logger.Error(err.Error())
		w.WriteHeader(400)
		return
	}
	data := newStorageUsersData(storageUsersSlice, src, offset)
	storageUser, _ := db.StorageUserGetByID(r.Context(), rc.dbpool, rc.userID)
	data.Caller = storageUser
	tmpl := template.Must(
		template.ParseFiles(
			"templates/users.html",
			"templates/users-assets.html",
		),
	).Lookup("users-search")
	tmpl.Execute(w, data)
}

func UserPutAPI(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	params httprouter.Params,
) {
	var userInput db.StorageUserInput
	err := common.BindJSON(r, &userInput)
	if err != nil {
		rc.logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	userInput.ID = params.ByName("id")
	if userInput.ID == rc.userID {
		rc.logger.Info("Cannot update self")
		common.ErrorResp(w, common.Forbidden)
		return
	}
	user, err := userInput.StorageUserBind()
	if err != nil {
		rc.logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		w.WriteHeader(400)
		return
	}
	if user.Role == db.Admin {
		rc.logger.Info("Can't set admin role")
		w.WriteHeader(403)
		return
	}
	err = rc.validate.StructPartial(user, "Active")
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), user)
		rc.logger.Info(err.Error())
		w.WriteHeader(400)
		return
	}
	err = db.StorageUserUpdate(r.Context(), rc.dbpool, user)
	if err != nil {
		errStruct := db.ErrorAsStruct(err)
		switch errStruct.(type) {
		case db.InvalidUUID, db.DoesNotExist:
			rc.logger.Info("Not found")
			common.ErrorResp(w, common.NotFound)
			w.WriteHeader(400)
			return
		default:
			rc.logger.Error(err.Error())
			common.ErrorResp(w, common.Internal)
			w.WriteHeader(500)
			return
		}
	}
}
