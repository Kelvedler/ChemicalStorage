package common

import (
	"net/http"
	"text/template"
)

const (
	NotFound     = "not-found"
	Unauthorized = "unauthorized"
	Forbidden    = "forbidden"
	Internal     = "internal"
)

type errorData struct {
	Title   string
	Message string
}

func ErrorResp(w http.ResponseWriter, reason string) {
	var data errorData
	switch reason {
	case NotFound:
		data.Title = "Не знайдено"
		data.Message = "Сторінка не знайдена"
	case Unauthorized:
		data.Title = "Не авторизований"
		data.Message = "Користувач не авторизований"
	case Forbidden:
		data.Title = "Заборонено"
		data.Message = "Відсутні права для використання даного ресурсу"
	default:
		data.Title = "Внутрішня помилка"
		data.Message = "Щось пішло не так"
	}
	w.Header().Set("HX-Retarget", "#content")
	tmpl := template.Must(template.ParseFiles("templates/base.html")).Lookup("error-page")
	tmpl.Execute(w, data)
}
