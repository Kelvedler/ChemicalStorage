package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Kelvedler/ChemicalStorage/pkg/auth"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
)

func PerformAuth(
	logger *slog.Logger,
	dbpool *pgxpool.Pool,
	w http.ResponseWriter,
	r *http.Request,
	required bool,
) (userID uuid.UUID, userRole db.Role, returnErr error) {
	accessToken, err := r.Cookie("access")
	if required {
		returnErr = errors.New("Unauthorized")
	}
	if err != nil {
		return userID, userRole, returnErr
	}

	tokenClaims, ok := auth.ValidateJWT(logger, accessToken.Value)
	if !ok {
		return userID, userRole, returnErr
	}

	userIDStr := tokenClaims[auth.ClaimSubject].(string)
	userRole, err = db.StringToRole(tokenClaims[auth.ClaimAccessRights].(string))
	if err != nil {
		return uuid.Nil, db.Role{}, err
	}
	if auth.RenewalAllowed(int64(tokenClaims[auth.ClaimIssuedAt].(float64))) {
		auth.ReissueTokenCookie(w, tokenClaims)
		return userID, userRole, nil
	}

	userID, err = uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, db.Role{}, returnErr
	}

	caller := db.StorageUser{ID: userID}
	errs := db.PerformBatch(r.Context(), dbpool, []db.BatchSet{caller.GetByID})
	userErr := errs[0]
	if userErr == nil {
		if caller.Active {
			auth.SetNewTokenCookie(w, caller)
			return userID, caller.Role, nil
		} else {
			return uuid.Nil, db.Role{}, returnErr
		}
	}

	errStruct := db.ErrorAsStruct(userErr)
	switch errStruct.(type) {
	case db.DoesNotExist:
		logger.Info("Not found")
		return uuid.Nil, db.Role{}, returnErr
	case db.ContextCanceled:
		logger.Warn("Context canceled")
		return uuid.Nil, db.Role{}, returnErr
	default:
		panic(fmt.Sprintf("unexpected err type, %t", errStruct))
	}
}
