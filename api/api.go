package api

import (
	"encoding/json"
	"net/http"
)

type APIRequest struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

type APIResponse struct {
	StatusCode  int         `json:"statusCode"`
	ResponseMsg string      `json:"responseMsg"`
	Data        interface{} `json:"data"`
}

func (a *APIResponse) ToString() (string, error) {
	bVal, err := json.Marshal(a)
	return string(bVal), err
}

func (a *APIResponse) GetValidStatusCode() int {
	if a.StatusCode == 0 {
		return http.StatusInternalServerError
	}
	return a.StatusCode
}
