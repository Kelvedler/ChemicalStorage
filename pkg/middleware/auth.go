package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

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
) (userID string, userRole db.Role, returnErr error) {
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

	userID = tokenClaims[auth.ClaimSubject].(string)
	userRole, err = db.StringToRole(tokenClaims[auth.ClaimAccessRights].(string))
	if err != nil {
		return "", db.Role{}, err
	}
	if auth.RenewalAllowed(int64(tokenClaims[auth.ClaimIssuedAt].(float64))) {
		auth.ReissueTokenCookie(w, tokenClaims)
		return userID, userRole, nil
	}

	storageUser, err := db.StorageUserGetByID(r.Context(), dbpool, userID)
	if err == nil {
		if storageUser.Active {
			auth.SetNewTokenCookie(w, storageUser)
			return userID, storageUser.Role, nil
		} else {
			return "", db.Role{}, returnErr
		}
	}

	errStruct := db.ErrorAsStruct(err)
	switch errStruct.(type) {
	case db.InvalidUUID, db.DoesNotExist:
		logger.Info("Not found")
		return "", db.Role{}, returnErr
	case db.ContextCanceled:
		logger.Warn("Context canceled")
		return "", db.Role{}, returnErr
	default:
		panic(fmt.Sprintf("unexpected err type, %t", errStruct))
	}
}
