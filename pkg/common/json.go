package common

import (
	"encoding/json"
	"io"
	"net/http"
)

func BindJSON(r *http.Request, i interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, i)
	if err != nil {
		return err
	}
	return nil
}
