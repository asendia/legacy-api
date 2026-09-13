package p

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTelegramWebhookRequiresSecret(t *testing.T) {
	t.Setenv("TELEGRAM_ENABLED", "true")
	t.Setenv("TELEGRAM_WEBHOOK_SECRET", strings.Repeat("s", 32))
	for _, secret := range []string{"", "wrong"} {
		r := httptest.NewRequest("POST", "/legacy-api-telegram-webhook", strings.NewReader(`{}`))
		r.Header.Set("X-Telegram-Bot-Api-Secret-Token", secret)
		w := httptest.NewRecorder()
		TelegramWebhook(w, r)
		if w.Code != 403 {
			t.Fatal("unauthenticated webhook accepted")
		}
	}
}

func TestTelegramWebhookIgnoresGroups(t *testing.T) {
	t.Setenv("TELEGRAM_ENABLED", "true")
	secret := strings.Repeat("s", 32)
	t.Setenv("TELEGRAM_WEBHOOK_SECRET", secret)
	r := httptest.NewRequest("POST", "/legacy-api-telegram-webhook", strings.NewReader(`{"message":{"text":"/start abc","from":{"id":123},"chat":{"id":-456,"type":"group"}}}`))
	r.Header.Set("X-Telegram-Bot-Api-Secret-Token", secret)
	w := httptest.NewRecorder()
	TelegramWebhook(w, r)
	if w.Code != 200 {
		t.Fatal("group update should be ignored")
	}
}

func TestTelegramDisabled(t *testing.T) {
	t.Setenv("TELEGRAM_ENABLED", "false")
	r := httptest.NewRequest("POST", "/legacy-api-telegram", strings.NewReader(`{"action":"login-start"}`))
	r.Header.Set("Origin", "https://sejiwo.com")
	w := httptest.NewRecorder()
	TelegramAPI(w, r)
	if w.Code != 404 {
		t.Fatal("disabled endpoint must be unavailable")
	}
	r.Header.Set("Authorization", "Bearer tg_"+strings.Repeat("s", 43))
	if _, err := VerifyNetlifyJWT(r); err == nil {
		t.Fatal("disabled Telegram session accepted")
	}
}
