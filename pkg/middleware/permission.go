package middleware

import (
	"errors"

	"github.com/Kelvedler/ChemicalStorage/pkg/db"
)

var (
	ErrorForbidden = errors.New("Request forbidden")
	AllowAll       = []db.RoleName{db.Admin, db.Lecturer, db.Assistant, db.Unconfirmed}
	AssistantOnly  = []db.RoleName{db.Assistant}
	AdminOnly      = []db.RoleName{db.Admin}
)

func CheckPermission(userRole string, allowedRoles []db.RoleName) error {
	for _, allowedRole := range allowedRoles {
		if userRole == string(allowedRole) {
			return nil
		}
	}
	return ErrorForbidden
}
