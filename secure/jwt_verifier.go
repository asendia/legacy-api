package secure

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type JWTAppMetaData struct {
	Provider string `json:"provider"`
}
type JWTUserMetadata struct {
	FullName string `json:"full_name"`
}
type JWTResponse struct {
	ID                 string          `json:"id"`
	Aud                string          `json:"aud"`
	Role               string          `json:"role"`
	Email              string          `json:"email"`
	ConfirmedAt        string          `json:"confirmed_at"`
	ConfirmationSentAt string          `json:"confirmation_sent_at"`
	AppMetadata        JWTAppMetaData  `json:"app_metadata"`
	UserMetadata       JWTUserMetadata `json:"user_metadata"`
	CreatedAt          string          `json:"created_at"`
	UpdatedAt          string          `json:"updated_at"`
}

type JWTResponseInvalid struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

func VerifyNetlifyJWT(c HTTPClient, authHeader string) (jwtRes JWTResponse, err error) {
	authHeaderSlice := strings.Split(authHeader, " ")
	if len(authHeaderSlice) != 2 || authHeaderSlice[0] != "Bearer" {
		return jwtRes, errors.New("Invalid authorization header")
	}
	req, err := http.NewRequest(http.MethodGet, "https://sejiwo.com/.netlify/identity/user", nil)
	if err != nil {
		return jwtRes, err
	}
	req.Header.Set("authorization", "Bearer "+authHeaderSlice[1])
	req.Header.Set("accept-encoding", "identity")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("referer", "https://sejiwo.com/")
	req.Header.Set("authority", "sejiwo.com")
	req.Header.Set("user-agent", "legacy-go-server")
	res, err := c.Do(req)
	if err != nil {
		return jwtRes, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		resData := JWTResponseInvalid{}
		err = json.NewDecoder(res.Body).Decode(&resData)
		if err != nil {
			return jwtRes, err
		}
		return jwtRes, errors.New(resData.Msg)
	}
	err = json.NewDecoder(res.Body).Decode(&jwtRes)
	if err != nil {
		return jwtRes, err
	}
	return jwtRes, err
}
