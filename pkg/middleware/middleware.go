package middleware

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/julienschmidt/httprouter"
	"github.com/microcosm-cc/bluemonday"

	"github.com/Kelvedler/ChemicalStorage/pkg/common"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
	"github.com/Kelvedler/ChemicalStorage/pkg/env"
)

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

type HandlerContext struct {
	dbpool   *pgxpool.Pool
	sanitize *bluemonday.Policy
	validate *validator.Validate
}

func NewHandlerContext(
	dbpool *pgxpool.Pool,
	sanitize *bluemonday.Policy,
	validate *validator.Validate,
) *HandlerContext {
	return &HandlerContext{
		dbpool:   dbpool,
		sanitize: sanitize,
		validate: validate,
	}
}

type Handle func(*RequestContext, http.ResponseWriter, *http.Request, httprouter.Params)

type RequestContext struct {
	Logger   *slog.Logger
	UserID   uuid.UUID
	UserRole db.Role
	DBpool   *pgxpool.Pool
	Sanitize *bluemonday.Policy
	Validate *validator.Validate
}

func (settings Settings) Wrapper(
	handler Handle,
	handlerContext *HandlerContext,
) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		logger := NewRequestLogger()
		RequestLogger(logger, r)
		if !HostInAllowed(r, env.Env.AllowedHosts) {
			common.ErrorResp(w, common.Forbidden)
			logger.Info(fmt.Sprintf("Host %s not allowed", r.Host))
			return
		}
		var userID uuid.UUID
		var userRole db.Role
		var err error
		if !settings.AuthExempt {
			userID, userRole, err = PerformAuth(
				logger,
				handlerContext.dbpool,
				w,
				r,
				settings.AuthRequired,
			)
			if err != nil {
				logger.Info(err.Error())
				common.ErrorResp(w, common.Unauthorized)
				return
			}

		}
		if !settings.XsrfExempt && !ValidateForXSRF(r, userID) {
			common.ErrorResp(w, common.Forbidden)
			logger.Warn("XSRF token invalid")
			return
		}
		if settings.AuthRequired {
			err = CheckPermission(userRole, settings.AllowedRoles)
			if err != nil {
				logger.Info(err.Error())
				common.ErrorResp(w, common.Forbidden)
				return
			}
		}
		rc := &RequestContext{
			Logger:   logger,
			UserID:   userID,
			UserRole: userRole,
			DBpool:   handlerContext.dbpool,
			Sanitize: handlerContext.sanitize,
			Validate: handlerContext.validate,
		}
		handler(rc, w, r, p)
	}
}
