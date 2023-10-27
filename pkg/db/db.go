package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func GetConnectionPool(ctx context.Context) *pgxpool.Pool {
	dbpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	return dbpool
}

const uniqueViolation = "23505"

type DBError struct {
	asMap    map[string]string
	asString string
}

func (err DBError) Error() string {
	return err.asString
}

func (err DBError) Map() map[string]string {
	return err.asMap
}

func LocalizeError(err error, tableStruct interface{}) error {
	var dbErr DBError
	dbErr.asMap = make(map[string]string)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolation:
			columnRe := regexp.MustCompile(fmt.Sprintf("%s_([a-z]+)_key", pgErr.TableName))
			column := columnRe.FindStringSubmatch(pgErr.ConstraintName)[1]
			column = cases.Title(language.English).String(column)
			reflection := reflect.TypeOf(tableStruct)
			reflectedField, _ := reflection.FieldByName(column)
			localColumn := reflectedField.Tag.Get("uaLocal")
			dbErr.asMap[column+"Err"] = fmt.Sprintf(
				"Елемент з даним %s уже існує",
				localColumn,
			)
		}
	}
	return dbErr
}
