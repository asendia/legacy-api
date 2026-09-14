package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSendText(t *testing.T) {
	client := Client{Token: "private-token", HTTPClient: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatal("expected POST")
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["text"] != "<private> & text" || body["chat_id"] != float64(123) || body["parse_mode"] != nil {
			t.Fatal("message changed")
		}
		if body["link_preview_options"].(map[string]interface{})["is_disabled"] != true {
			t.Fatal("link preview must be disabled")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})}}
	if err := client.SendText(context.Background(), 123, "<private> & text"); err != nil {
		t.Fatal(err)
	}
}

func TestSendFailure(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		network bool
	}{
		{"limit", 429, `{"ok":false,"error_code":429,"parameters":{"retry_after":30}}`, false},
		{"blocked", 403, `{"ok":false,"error_code":403}`, false},
		{"invalid", 200, `private-token`, false},
		{"network", 0, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := Client{Token: "private-token", HTTPClient: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				if tc.network {
					return nil, errors.New(r.URL.String())
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})}}
			err := client.SendText(context.Background(), 123, "private text")
			if err == nil || strings.Contains(err.Error(), "private-token") || strings.Contains(err.Error(), "private text") {
				t.Fatal("unsafe or missing error")
			}
			if tc.name == "limit" {
				var sendError *SendError
				if !errors.As(err, &sendError) || sendError.RetryAfter != 30*time.Second {
					t.Fatal("missing retry delay")
				}
			}
		})
	}
}

func TestRejectInvalidDestination(t *testing.T) {
	client := Client{Token: "token"}
	for _, id := range []int64{0, -123} {
		if client.SendText(context.Background(), id, "message") == nil {
			t.Fatal("group and empty chats must be rejected")
		}
	}
}
