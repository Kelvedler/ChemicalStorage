package db

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	uniqueViolation           = "23505"
	invalidTextRepresentation = "22P02"
)

type DBError struct {
	asMapLocal map[string]string
	asString   string
}

func (err DBError) Error() string {
	return err.asString
}

func (err DBError) Map() map[string]string {
	return err.asMapLocal
}

type UniqueViolation struct {
	table  string
	column string
}

type InvalidUUID struct{}

type DoesNotExist struct{}

func ErrorAsStruct(err error) interface{} {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolation:
			columnRe := regexp.MustCompile(fmt.Sprintf("%s_([a-z]+)_key", pgErr.TableName))
			column := columnRe.FindStringSubmatch(pgErr.ConstraintName)[1]
			column = cases.Title(language.English).String(column)
			return UniqueViolation{
				table:  pgErr.TableName,
				column: column,
			}
		case invalidTextRepresentation:
			return InvalidUUID{}
		default:
			panic(fmt.Sprintf("unforseen case - %s code", pgErr.Code))
		}
	} else {
		switch err {
		case pgx.ErrNoRows:
			return DoesNotExist{}
		default:
			panic(fmt.Sprintf("unforseen case - %s", err))
		}
	}
}

func (u UniqueViolation) LocalizeUniqueViolation(tableStruct interface{}) error {
	reflection := reflect.TypeOf(tableStruct)
	reflectedField, _ := reflection.FieldByName(u.column)
	localColumn := reflectedField.Tag.Get("uaLocal")
	var dbErr DBError
	dbErr.asMapLocal = make(map[string]string)
	dbErr.asString = fmt.Sprintf("%s with given %s exists", u.table, u.column)
	dbErr.asMapLocal[u.column+"Err"] = fmt.Sprintf(
		"Елемент з даним %s уже існує",
		localColumn,
	)
	return dbErr
}
