package api

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/asendia/legacy-api/mail"
	"github.com/asendia/legacy-api/telegram"
)

func telegramTest(t *testing.T) (APIForScheduler, MessageData) {
	t.Helper()
	t.Setenv("TELEGRAM_ENABLED", "true")
	t.Setenv("SERVERLESS_FUNCTION_SOURCE_CODE", "../mail/")
	t.Setenv("ENCRYPTION_KEY", "01234567890123456789012345678901")
	ctx := context.Background()
	tx, err := pgxPoolConn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tx.Rollback(ctx) })
	migration, err := os.ReadFile("../data/migrations/001_telegram.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ReplaceAll(strings.ReplaceAll(string(migration), "BEGIN;", ""), "COMMIT;", "")
	if _, err = tx.Exec(ctx, sql); err != nil {
		t.Fatal(err)
	}
	msg := generateMessageTemplate()
	writer := APIForFrontend{Context: ctx, Tx: tx}
	res, err := writer.InsertMessage(generateJwtMessageTemplate(msg.EmailCreator), APIParamInsertMessage{EmailReceivers: msg.EmailReceivers, MessageContent: "aes.utf8:private-content", InactivePeriodDays: 60, ReminderIntervalDays: 15})
	if err != nil {
		t.Fatal(err)
	}
	saved := res.Data.(MessageData)
	if _, err := tx.Exec(ctx, `INSERT INTO telegram_accounts(email,subject,chat_id,phone_hash,reminders_enabled) VALUES ($1,'subject',123,'phone',true)`, msg.EmailCreator); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO telegram_receivers(message_id,email_receiver,token_hash,chat_id) VALUES ($1,$2,'link',456)`, saved.ID, msg.EmailReceivers[0]); err != nil {
		t.Fatal(err)
	}
	return APIForScheduler{Context: ctx, Tx: tx}, saved
}

func TestTelegramRetryAfterEmailSuccess(t *testing.T) {
	a, msg := telegramTest(t)
	if _, err := a.Tx.Exec(a.Context, `UPDATE messages SET inactive_at=CURRENT_DATE-1 WHERE id=$1`, msg.ID); err != nil {
		t.Fatal(err)
	}
	a.SendEmail = func(items []mail.MailItem) []mail.SendEmailsResponse {
		result := make([]mail.SendEmailsResponse, len(items))
		return result
	}
	if _, err := a.SendTestamentsOfInactiveMessages(); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := a.Tx.QueryRow(a.Context, `SELECT sent_counter FROM messages WHERE id=$1`, msg.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one completed cycle, got %d", count)
	}
	a.SendTelegram = func(context.Context, int64, string) error { return &telegram.SendError{Code: 429} }
	if err := a.SendPendingTelegram(); err != nil {
		t.Fatal(err)
	}
	var attempts int
	if err := a.Tx.QueryRow(a.Context, `SELECT attempts FROM telegram_deliveries`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatal("expected stored retry")
	}
	if _, err := a.Tx.Exec(a.Context, `UPDATE telegram_deliveries SET next_attempt_at=now()`); err != nil {
		t.Fatal(err)
	}
	sends := 0
	a.SendTelegram = func(_ context.Context, chat int64, body string) error {
		sends++
		if chat != 456 || !strings.Contains(body, "aes.utf8:private-content") {
			t.Fatal("wrong delivery")
		}
		return nil
	}
	if err := a.SendPendingTelegram(); err != nil {
		t.Fatal(err)
	}
	if err := a.SendPendingTelegram(); err != nil {
		t.Fatal(err)
	}
	if sends != 1 {
		t.Fatal("successful delivery was repeated")
	}
	var body string
	if err := a.Tx.QueryRow(a.Context, `SELECT body_encrypted FROM telegram_deliveries`).Scan(&body); err != nil {
		t.Fatal(err)
	}
	if body != "" {
		t.Fatal("sent body must be removed")
	}
}

func TestTelegramCancelPending(t *testing.T) {
	for _, reason := range []string{"extend", "unsubscribe", "remove", "stop"} {
		t.Run(reason, func(t *testing.T) {
			a, msg := telegramTest(t)
			if _, err := a.Tx.Exec(a.Context, `UPDATE messages SET inactive_at=CURRENT_DATE-1 WHERE id=$1`, msg.ID); err != nil {
				t.Fatal(err)
			}
			if err := a.queueTelegram("final"); err != nil {
				t.Fatal(err)
			}
			var err error
			switch reason {
			case "extend":
				_, err = a.Tx.Exec(a.Context, `UPDATE messages SET extension_secret=repeat('a',69) WHERE id=$1`, msg.ID)
			case "unsubscribe":
				_, err = a.Tx.Exec(a.Context, `UPDATE messages_email_receivers SET is_unsubscribed=true WHERE message_id=$1`, msg.ID)
			case "remove":
				_, err = a.Tx.Exec(a.Context, `DELETE FROM telegram_receivers WHERE message_id=$1`, msg.ID)
			case "stop":
				_, err = a.Tx.Exec(a.Context, `UPDATE telegram_receivers SET chat_id=NULL WHERE message_id=$1`, msg.ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			a.SendTelegram = func(context.Context, int64, string) error { t.Fatal("cancelled delivery was sent"); return nil }
			if err := a.SendPendingTelegram(); err != nil {
				t.Fatal(err)
			}
			var stopped bool
			if err := a.Tx.QueryRow(a.Context, `SELECT stopped_at IS NOT NULL FROM telegram_deliveries`).Scan(&stopped); err != nil {
				t.Fatal(err)
			}
			if !stopped {
				t.Fatal("delivery must stop")
			}
		})
	}
}

func TestTelegramQueueDoesNotRepeat(t *testing.T) {
	a, msg := telegramTest(t)
	if _, err := a.Tx.Exec(a.Context, `UPDATE messages SET next_reminder_at=CURRENT_DATE-1 WHERE id=$1`, msg.ID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := a.queueTelegram("reminder"); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := a.Tx.QueryRow(a.Context, `SELECT count(*) FROM telegram_deliveries`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("duplicate reminder")
	}
	if _, err := a.Tx.Exec(a.Context, `UPDATE messages SET inactive_at=CURRENT_DATE-1 WHERE id=$1`, msg.ID); err != nil {
		t.Fatal(err)
	}
	a.SendTelegram = func(context.Context, int64, string) error { t.Fatal("expired reminder was sent"); return nil }
	if err := a.SendPendingTelegram(); err != nil {
		t.Fatal(err)
	}
}

func TestEmailFailureKeepsRecipientActive(t *testing.T) {
	a, msg := telegramTest(t)
	if _, err := a.Tx.Exec(a.Context, `UPDATE messages SET inactive_at=CURRENT_DATE-1 WHERE id=$1`, msg.ID); err != nil {
		t.Fatal(err)
	}
	failedEmail := ""
	a.SendEmail = func(items []mail.MailItem) []mail.SendEmailsResponse {
		failedEmail = items[0].To[0].Email
		return []mail.SendEmailsResponse{{Err: errors.New("temporary failure")}, {}}
	}
	if _, err := a.SendTestamentsOfInactiveMessages(); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := a.Tx.QueryRow(a.Context, `SELECT sent_counter FROM messages WHERE id=$1`, msg.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("partial cycle must remain open")
	}
	var inactive int
	if err := a.Tx.QueryRow(a.Context, `SELECT count(*) FROM emails WHERE NOT is_active`).Scan(&inactive); err != nil {
		t.Fatal(err)
	}
	if inactive != 0 {
		t.Fatal("temporary failure disabled a recipient")
	}
	a.SendEmail = func(items []mail.MailItem) []mail.SendEmailsResponse {
		if len(items) != 1 || items[0].To[0].Email != failedEmail {
			t.Fatal("partial retry repeated a successful email")
		}
		return []mail.SendEmailsResponse{{}}
	}
	if _, err := a.SendTestamentsOfInactiveMessages(); err != nil {
		t.Fatal(err)
	}
	if err := a.Tx.QueryRow(a.Context, `SELECT sent_counter FROM messages WHERE id=$1`, msg.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("completed retry must advance one cycle")
	}
}
