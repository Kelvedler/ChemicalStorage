package view

import (
	"fmt"
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

func sanitizeStorageUserShort(
	rc *RequestContext,
	storageUser *db.StorageUserShort,
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
	var storageUserInput db.StorageUserShort
	err := common.BindJSON(r, &storageUserInput)
	if err != nil {
		rc.logger.Error(err.Error())
		common.DefaultErrorResp(w)
		return
	}
	sanitizeStorageUserShort(rc, &storageUserInput)
	tmpl := template.Must(template.ParseFiles("templates/sign-in-assets.html")).
		Lookup("sign-in-form")

	errMap := make(map[string]string)
	errMap["InputErr"] = "Невірний логін або пароль"
	errMap["Name"] = storageUserInput.Name
	errMap["Password"] = storageUserInput.Password
	err = rc.validate.Struct(storageUserInput)
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
			return
		default:
			panic(fmt.Sprintf("unexpected err type, %t", errStruct))
		}
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
	err = auth.SetNewTokenCookie(w, storageUser)
	if err != nil {
		rc.logger.Error(err.Error())
		common.DefaultErrorResp(w)
		return
	}

	w.Header().Set("HX-Redirect", "/me")
}
