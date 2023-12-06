package view

import (
	"html/template"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/julienschmidt/httprouter"

	"github.com/Kelvedler/ChemicalStorage/pkg/auth"
	"github.com/Kelvedler/ChemicalStorage/pkg/common"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
	"github.com/Kelvedler/ChemicalStorage/pkg/middleware"
)

func SignUp(
	_ *middleware.RequestContext,
	w http.ResponseWriter,
	_ *http.Request,
	_ httprouter.Params,
) {
	tmpl := template.Must(
		template.ParseFiles(
			"templates/sign-up.html",
			"templates/sign-up-assets.html",
			"templates/base.html",
		),
	)
	tmpl.Execute(w, nil)
}

type signUpInput struct {
	Name      string `json:"name"`
	Password1 string `json:"password_1"`
	Password2 string `json:"password_2"`
}

func sanitizeSignUpInput(
	rc *middleware.RequestContext,
	input *signUpInput,
) {
	sanitizer := rc.Sanitize
	input.Name = sanitizer.Sanitize(input.Name)
	input.Password1 = sanitizer.Sanitize(input.Password1)
	input.Password2 = sanitizer.Sanitize(input.Password2)
}

func signUpErrMapAddInput(errMap map[string]string, input signUpInput) {
	errMap["Name"] = input.Name
	errMap["Password1"] = input.Password1
	errMap["Password2"] = input.Password2
}

func SignUpAPI(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	var input signUpInput
	err := common.BindJSON(r, &input)
	if err != nil {
		rc.Logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	sanitizeSignUpInput(rc, &input)
	tmpl := template.Must(template.ParseFiles("templates/sign-up-assets.html")).
		Lookup("sign-up-form")
	if input.Password1 != input.Password2 {
		errMap := make(map[string]string)
		signUpErrMapAddInput(errMap, input)
		errMap["PasswordErr"] = "Паролі не співпадають"
		tmpl.Execute(w, errMap)
		return
	}
	newStorageUser := db.StorageUser{
		Name:     input.Name,
		Password: input.Password1,
		Role:     db.Unconfirmed,
	}
	err = rc.Validate.StructPartial(newStorageUser, "Name", "Password")
	if err != nil {
		err = common.LocalizeValidationErrors(err.(validator.ValidationErrors), newStorageUser)
		rc.Logger.Info(err.Error())
		errMap := err.(common.ValidationError).Map()
		signUpErrMapAddInput(errMap, input)
		tmpl.Execute(w, errMap)
		return
	}
	hashedPassword, err := common.HashPassword([]byte(newStorageUser.Password))
	if err != nil {
		rc.Logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}
	newStorageUser.Password = hashedPassword
	errs := db.PerformBatch(r.Context(), rc.DBpool, []db.BatchSet{newStorageUser.Create})
	userErr := errs[0]
	if userErr != nil {
		errStruct := db.ErrorAsStruct(userErr)
		switch errStruct.(type) {
		case db.UniqueViolation:
			err = errStruct.(db.UniqueViolation).Localize(newStorageUser)
			rc.Logger.Info(err.Error())
			errMap := err.(db.DBError).Map()
			signUpErrMapAddInput(errMap, input)
			tmpl.Execute(w, errMap)
			return
		default:
			rc.Logger.Error(userErr.Error())
			common.ErrorResp(w, common.Internal)
			return
		}
	}
	err = auth.SetNewTokenCookie(w, newStorageUser)
	if err != nil {
		rc.Logger.Error(err.Error())
		common.ErrorResp(w, common.Internal)
		return
	}

	w.Header().Set("HX-Redirect", "/me")
}
