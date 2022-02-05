package api

import (
	"encoding/json"
)

type APIRequest struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

type APIRequestMessageData struct {
	Action string      `json:"action"`
	Data   MessageData `json:"data"`
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
