package db

import (
	"context"
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
	outOfLimits               = "A0001"
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

type OutOfLimits struct {
	table  string
	column string
}

type ContextCanceled struct{}

func getColumn(pgErr *pgconn.PgError) string {
	columnRe := regexp.MustCompile(fmt.Sprintf("%s_([a-z]+).+", pgErr.TableName))
	column := columnRe.FindStringSubmatch(pgErr.ConstraintName)[1]
	return cases.Title(language.English).String(column)
}

func ErrorAsStruct(err error) interface{} {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolation:
			return UniqueViolation{
				table:  pgErr.TableName,
				column: getColumn(pgErr),
			}
		case invalidTextRepresentation:
			return InvalidUUID{}
		case outOfLimits:
			return OutOfLimits{
				table:  pgErr.TableName,
				column: getColumn(pgErr),
			}
		default:
			panic(fmt.Sprintf("unforseen case - %s code", pgErr.Code))
		}
	} else {
		switch err {
		case pgx.ErrNoRows:
			return DoesNotExist{}
		case context.Canceled:
			return ContextCanceled{}
		default:
			panic(fmt.Sprintf("unforseen case - %s", err))
		}
	}
}

func localColumn(column string, tableStruct interface{}) string {
	reflection := reflect.TypeOf(tableStruct)
	reflectedField, _ := reflection.FieldByName(column)
	return reflectedField.Tag.Get("uaLocal")
}

func (u UniqueViolation) Localize(tableStruct interface{}) error {
	var dbErr DBError
	dbErr.asMapLocal = make(map[string]string)
	dbErr.asString = fmt.Sprintf("%s with given %s exists", u.table, u.column)
	dbErr.asMapLocal[u.column+"Err"] = fmt.Sprintf(
		"Елемент з даним %s уже існує",
		localColumn(u.column, tableStruct),
	)
	return dbErr
}

func (o OutOfLimits) Localize(tableStruct interface{}) error {
	var dbErr DBError
	dbErr.asMapLocal = make(map[string]string)
	dbErr.asString = fmt.Sprintf("%s out of limits for %s", o.column, o.table)
	dbErr.asMapLocal[o.column+"Err"] = fmt.Sprintf(
		"%s поза межами",
		localColumn(o.column, tableStruct),
	)
	return dbErr
}
