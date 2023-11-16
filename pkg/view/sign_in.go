package view

import (
	"html/template"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/julienschmidt/httprouter"

	"github.com/Kelvedler/ChemicalStorage/pkg/auth"
	"github.com/Kelvedler/ChemicalStorage/pkg/common"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
)

func SignIn(
	_ *RequestContext,
	w http.ResponseWriter,
	_ *http.Request,
	_ httprouter.Params,
) {
	tmpl := template.Must(
		template.ParseFiles(
			"templates/sign-in.html",
			"templates/base.html",
			"templates/sign-in-assets.html",
		),
	)
	tmpl.Execute(w, nil)
}

func sanitizeStorageUser(
	rc *RequestContext,
	storageUser *db.StorageUser,
) {
	sanitizer := rc.sanitize
	storageUser.Name = sanitizer.Sanitize(storageUser.Name)
	storageUser.Password = sanitizer.Sanitize(storageUser.Password)
}

func SignInAPI(
	rc *RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	var storageUserInput db.StorageUser
	err := common.BindJSON(r, &storageUserInput)
	if err != nil {
		rc.logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	sanitizeStorageUser(rc, &storageUserInput)
	tmpl := template.Must(template.ParseFiles("templates/sign-in-assets.html")).
		Lookup("sign-in-form")

	errMap := make(map[string]string)
	errMap["InputErr"] = "Невірний логін або пароль"
	errMap["Name"] = storageUserInput.Name
	errMap["Password"] = storageUserInput.Password
	err = rc.validate.StructPartial(storageUserInput, "Name", "Password")
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), storageUserInput)
		rc.logger.Info(err.Error())
		tmpl.Execute(w, errMap)
		return
	}
	storageUser, err := db.StorageUserGetByName(
		r.Context(),
		rc.dbpool,
		storageUserInput.Name,
	)
	if err != nil {
		errStruct := db.ErrorAsStruct(err)
		switch errStruct.(type) {
		case db.DoesNotExist:
			rc.logger.Info("Not found")
			tmpl.Execute(w, errMap)
		default:
			rc.logger.Error(err.Error())
			common.ErrorResp(w, common.Internal)
		}
		return
	}
	passwordCorrect, err := common.ComparePasswords(storageUserInput.Password, storageUser.Password)
	if !passwordCorrect {
		if err == nil {
			rc.logger.Info("Invalid password")
		} else {
			rc.logger.Error(err.Error())
		}
		tmpl.Execute(w, errMap)
	}
	if !storageUser.Active {
		rc.logger.Info("Deactivated user")
		errMap["InputErr"] = "Акаунт декативовано"
		tmpl.Execute(w, errMap)
		return
	}
	err = auth.SetNewTokenCookie(w, storageUser)
	if err != nil {
		rc.logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}

	w.Header().Set("HX-Redirect", "/me")
}
