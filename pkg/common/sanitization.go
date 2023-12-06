package common

import (
	"github.com/microcosm-cc/bluemonday"
)

func NewSanitizer() *bluemonday.Policy {
	return bluemonday.UGCPolicy()
}
