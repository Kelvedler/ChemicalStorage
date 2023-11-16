package auth

import (
	"errors"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt"

	"github.com/Kelvedler/ChemicalStorage/pkg/db"
	"github.com/Kelvedler/ChemicalStorage/pkg/env"
)

const (
	ClaimSubject        = "sub"
	ClaimAccessRights   = "acr"
	ClaimIssuedAt       = "iat"
	ClaimExpirationTime = "exp"
)

var (
	ErrSubject    = errors.New("Invalid JWT subject")
	ErrUnexpected = errors.New("Unexpected validation error")
	ErrAccess     = errors.New("Subject has no access right")
	ErrIssued     = errors.New("Invalid JWT issued at")
	ErrExpiration = errors.New("Token expired")
)

func JWTExpiration() time.Time {
	return time.Now().Add(
		time.Duration(env.Env.Jwt.ExpirationDeltaMinutes) * time.Minute,
	).UTC()
}

func RenewalAllowed(issuedAt int64) bool {
	issuedExp := env.Env.Jwt.ExpirationDeltaMinutes * 60
	nowUnix := time.Now().Unix()
	expUnix := time.Unix(int64(issuedExp), int64(0)).Unix() + issuedAt
	return nowUnix > expUnix
}

func IssueJWT(user db.StorageUser) (string, error) {
	claims := jwt.MapClaims{}
	claims[ClaimSubject] = user.ID.String()
	claims[ClaimAccessRights] = user.Role.Name
	claims[ClaimIssuedAt] = time.Now().Unix()
	claims[ClaimExpirationTime] = JWTExpiration().Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(env.Env.SecretKey))
}

func ReissueJWT(claims jwt.MapClaims) (string, error) {
	claims[ClaimExpirationTime] = JWTExpiration().Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(env.Env.SecretKey))
}

func ValidateJWT(
	logger *slog.Logger,
	tokenString string,
) (jwt.MapClaims, bool) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte(env.Env.SecretKey), nil
	})
	if err != nil {
		logger.Info(err.Error())
		return jwt.MapClaims{}, false
	}
	tokenClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		logger.Error(ErrUnexpected.Error())
		return jwt.MapClaims{}, false
	}
	_, ok = tokenClaims[ClaimSubject].(string)
	if !ok {
		logger.Error(ErrSubject.Error())
		return jwt.MapClaims{}, false
	}
	_, ok = tokenClaims[ClaimAccessRights].(string)
	if !ok {
		logger.Info(ErrAccess.Error())
		return jwt.MapClaims{}, false
	}
	_, ok = tokenClaims[ClaimIssuedAt].(float64)
	if !ok {
		logger.Error(ErrIssued.Error())
		return jwt.MapClaims{}, false
	}
	tsNow := time.Now().Unix()
	expiredAt, ok := tokenClaims[ClaimExpirationTime].(float64)
	if !ok || tsNow > int64(expiredAt) {
		logger.Info(ErrExpiration.Error())
		return jwt.MapClaims{}, false
	}

	return tokenClaims, true
}
