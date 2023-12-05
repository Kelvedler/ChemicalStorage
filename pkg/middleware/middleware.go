package middleware

import "github.com/Kelvedler/ChemicalStorage/pkg/db"

type Settings struct {
	AuthRequired bool
	AuthExempt   bool
	AllowedRoles []db.Role
	XsrfExempt   bool
}

var Unrestricted = Settings{
	AuthRequired: false,
	AuthExempt:   false,
	AllowedRoles: AllowAll,
	XsrfExempt:   true,
}

var UnrestrictedNoAuth = Settings{
	AuthRequired: false,
	AuthExempt:   false,
	AllowedRoles: AllowAll,
	XsrfExempt:   true,
}

var LecturerAssistantView = Settings{
	AuthRequired: true,
	AuthExempt:   false,
	AllowedRoles: LecturerAssistant,
	XsrfExempt:   true,
}

var AssistantOnlyView = Settings{
	AuthRequired: true,
	AuthExempt:   false,
	AllowedRoles: AssistantOnly,
	XsrfExempt:   true,
}

var AdminOnlyView = Settings{
	AuthRequired: true,
	AuthExempt:   false,
	AllowedRoles: AdminOnly,
	XsrfExempt:   true,
}

var AssistantOnlyAPI = Settings{
	AuthRequired: true,
	AuthExempt:   false,
	AllowedRoles: AssistantOnly,
	XsrfExempt:   false,
}

var AssistantOnlyNoXsrf = Settings{
	AuthRequired: true,
	AuthExempt:   false,
	AllowedRoles: AssistantOnly,
	XsrfExempt:   true,
}

var AdminOnlyAPI = Settings{
	AuthRequired: true,
	AuthExempt:   false,
	AllowedRoles: AdminOnly,
	XsrfExempt:   false,
}
