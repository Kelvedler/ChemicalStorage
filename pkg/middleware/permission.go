package middleware

import (
	"errors"

	"github.com/Kelvedler/ChemicalStorage/pkg/db"
)

var (
	ErrorForbidden = errors.New("Request forbidden")
	AllowAll       = []db.Role{db.Admin, db.Lecturer, db.Assistant, db.Unconfirmed}
	AssistantOnly  = []db.Role{db.Assistant}
	AdminOnly      = []db.Role{db.Admin}
)

func CheckPermission(userRole string, allowedRoles []db.Role) error {
	for _, allowedRole := range allowedRoles {
		if userRole == allowedRole.Name {
			return nil
		}
	}
	return ErrorForbidden
}
