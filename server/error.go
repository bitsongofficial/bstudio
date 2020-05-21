package server

import (
	"encoding/json"
	"net/http"
)

func writeJSONResponse(rw http.ResponseWriter, code int, result interface{}) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(code)

	json.NewEncoder(rw).Encode(result)
}

type ErrorJsonBody struct {
	Message string `json:"message"`
}

type ErrorJson struct {
	Error ErrorJsonBody `json:"error"`
}

func newErrorJson(message string) ErrorJson {
	return ErrorJson{
		Error: ErrorJsonBody{
			Message: message,
		},
	}
}
