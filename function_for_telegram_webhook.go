package p

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/asendia/legacy-api/telegram"
	"github.com/jackc/pgx/v5"
)

func TelegramWebhook(w http.ResponseWriter, r *http.Request) {
	secret := os.Getenv("TELEGRAM_WEBHOOK_SECRET")
	if !telegramEnabled() || len(secret) < 32 || r.Method != http.MethodPost || subtle.ConstantTimeCompare([]byte(secret), []byte(r.Header.Get("X-Telegram-Bot-Api-Secret-Token"))) != 1 {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 65536)
	var update struct {
		Message struct {
			Text string `json:"text"`
			From struct {
				ID    int64 `json:"id"`
				IsBot bool  `json:"is_bot"`
			} `json:"from"`
			Chat struct {
				ID   int64  `json:"id"`
				Type string `json:"type"`
			} `json:"chat"`
		} `json:"message"`
	}
	if json.NewDecoder(r.Body).Decode(&update) != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	msg := update.Message
	if msg.Chat.Type != "private" || msg.From.ID != msg.Chat.ID || msg.Chat.ID <= 0 || msg.From.IsBot {
		w.WriteHeader(200)
		return
	}
	reply := ""
	err := telegramTransaction(r.Context(), func(tx pgx.Tx) error {
		ctx := r.Context()
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(719282)`); err != nil {
			return err
		}
		if msg.Text == "/stop" {
			reply = "Telegram delivery is stopped. Email delivery is unchanged."
			if _, err := tx.Exec(ctx, `UPDATE telegram_receivers SET chat_id=NULL WHERE chat_id=$1`, msg.Chat.ID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE telegram_accounts SET reminders_enabled=false WHERE chat_id=$1`, msg.Chat.ID); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `UPDATE telegram_deliveries SET stopped_at=now(),body_encrypted='' WHERE chat_id=$1 AND sent_at IS NULL`, msg.Chat.ID)
			return err
		}
		parts := strings.Fields(msg.Text)
		if len(parts) != 2 || parts[0] != "/start" || len(parts[1]) != 43 {
			return nil
		}
		tag, err := tx.Exec(ctx, `UPDATE telegram_receivers t SET chat_id=$1 FROM messages_email_receivers r WHERE t.token_hash=$2 AND (t.chat_id IS NULL OR t.chat_id=$1) AND r.message_id=t.message_id AND r.email_receiver=t.email_receiver AND NOT r.is_unsubscribed`, msg.Chat.ID, telegram.Hash(parts[1]))
		reply = "This link is no longer available. Ask the writer for a new private link."
		if err == nil && tag.RowsAffected() == 1 {
			reply = "Telegram delivery is connected. To stop Telegram delivery, send /stop."
		}
		return err
	})
	if err != nil {
		http.Error(w, "Try again", 500)
		return
	}
	if reply != "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"method": "sendMessage", "chat_id": msg.Chat.ID, "text": reply})
		return
	}
	w.WriteHeader(200)
}
