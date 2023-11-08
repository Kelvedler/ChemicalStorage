package auth

import "github.com/Kelvedler/ChemicalStorage/pkg/db"

var AllowAll = []string{db.RoleAdmin, db.RoleLecturer, db.RoleAssistant, db.RoleUnconfirmed}
