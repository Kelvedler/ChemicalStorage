package middleware

import (
	"errors"

	"github.com/Kelvedler/ChemicalStorage/pkg/db"
)

var (
	ErrorForbidden    = errors.New("Request forbidden")
	AllowAll          = []db.Role{db.Admin, db.Lecturer, db.Assistant, db.Unconfirmed}
	LecturerAssistant = []db.Role{db.Lecturer, db.Assistant}
	AssistantOnly     = []db.Role{db.Assistant}
	AdminOnly         = []db.Role{db.Admin}
)

func CheckPermission(userRole db.Role, allowedRoles []db.Role) error {
	for _, allowedRole := range allowedRoles {
		if userRole == allowedRole {
			return nil
		}
	}
	return ErrorForbidden
}
