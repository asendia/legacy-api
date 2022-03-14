package secure

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestVerifyJWTSuccess(t *testing.T) {
	token := "Bearer ValidToken4"
	mockC := &HTTPClientMock{
		Res: generateResMock(200, `{"id":"abc","aud":"","role":"","email":"test@sejiwo.com","confirmed_at":"2022-02-08T00:00:00Z","confirmation_sent_at":"2022-02-08T00:00:00Z","app_metadata":{"provider":"email"},"user_metadata":{"full_name":"Sir Legacy"},"created_at":"2022-02-08T00:00:00Z","updated_at":"2022-02-08T00:00:00Z"}`),
	}
	res, err := VerifyNetlifyJWT(mockC, token)
	if err != nil {
		t.Fatalf("JWT verification is failed, %v", err)
	}
	t.Logf("JWT %s is verified: %v", token, res)
}

func TestVerifyJWTFailed(t *testing.T) {
	mockC := &HTTPClientMock{
		Res: generateResMock(401, `{"code":401,"msg":"Invalid token: token contains an invalid number of segments"}`),
	}
	res, err := VerifyNetlifyJWT(mockC, "Bearer InvalidToken")
	if err == nil {
		t.Fatalf("JWT verification should be failed because the token is invalid")
	}
	t.Logf("JWT verification is failed as expected: %v %v", res, err)
}

func generateResMock(statusCode int, json string) *http.Response {
	body := ioutil.NopCloser(bytes.NewReader([]byte(json)))
	return &http.Response{
		StatusCode: statusCode,
		Body:       body,
	}
}

type HTTPClientMock struct {
	Res *http.Response
}

func (c *HTTPClientMock) Do(req *http.Request) (*http.Response, error) {
	return c.Res, nil
}
