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
) (userID, userRole string, returnErr error) {
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
	userRole = tokenClaims[auth.ClaimAccessRights].(string)
	if auth.RenewalAllowed(int64(tokenClaims[auth.ClaimIssuedAt].(float64))) {
		auth.ReissueTokenCookie(w, tokenClaims)
		return userID, userRole, nil
	}

	storageUser, err := db.StorageUserGetByID(r.Context(), dbpool, userID)
	if err == nil {
		auth.SetNewTokenCookie(w, storageUser)
		return userID, userRole, nil
	}

	errStruct := db.ErrorAsStruct(err)
	switch errStruct.(type) {
	case db.InvalidUUID, db.DoesNotExist:
		logger.Info("Not found")
		return "", "", returnErr
	case db.ContextCanceled:
		logger.Warn("Context canceled")
		return "", "", returnErr
	default:
		panic(fmt.Sprintf("unexpected err type, %t", errStruct))
	}
}
