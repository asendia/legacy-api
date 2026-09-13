package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	Token      string
	HTTPClient *http.Client
}

type SendError struct {
	Code       int
	RetryAfter time.Duration
}

func (e *SendError) Error() string {
	return "Telegram delivery failed (code " + strconv.Itoa(e.Code) + ")"
}

// SendText sends plain text. Errors do not contain the token or message.
func (c *Client) SendText(ctx context.Context, chatID int64, text string) error {
	if c.Token == "" || chatID <= 0 || text == "" || len([]rune(text)) > 4096 {
		return errors.New("invalid Telegram delivery settings")
	}
	body, err := json.Marshal(struct {
		ChatID             int64           `json:"chat_id"`
		Text               string          `json:"text"`
		LinkPreviewOptions map[string]bool `json:"link_preview_options"`
	}{chatID, text, map[string]bool{"is_disabled": true}})
	if err != nil {
		return errors.New("cannot encode Telegram message")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.telegram.org/bot"+c.Token+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return errors.New("cannot create Telegram request")
	}
	req.Header.Set("Content-Type", "application/json")
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	// Do not follow redirects that could expose the token or message.
	copyClient := *client
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	res, err := copyClient.Do(req)
	if err != nil {
		return errors.New("Telegram request failed")
	}
	defer res.Body.Close()
	var result struct {
		OK         bool `json:"ok"`
		ErrorCode  int  `json:"error_code"`
		Parameters struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 65536)).Decode(&result); err != nil {
		return errors.New("invalid Telegram response")
	}
	if res.StatusCode != http.StatusOK || !result.OK {
		code := result.ErrorCode
		if code == 0 {
			code = res.StatusCode
		}
		return &SendError{Code: code, RetryAfter: time.Duration(result.Parameters.RetryAfter) * time.Second}
	}
	return nil
}
