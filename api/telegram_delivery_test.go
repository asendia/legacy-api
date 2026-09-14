package api

import (
	"context"
	"errors"
	"fmt"
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
	for _, name := range []string{"001_telegram.sql", "002_email_retries.sql"} {
		migration, err := os.ReadFile("../data/migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		sql := strings.ReplaceAll(strings.ReplaceAll(string(migration), "BEGIN;", ""), "COMMIT;", "")
		if _, err = tx.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
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
	if _, err := a.Tx.Exec(a.Context, `UPDATE email_delivery_receipts SET next_attempt_at=now()`); err != nil {
		t.Fatal(err)
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

func TestFailedRecipientsDoNotBlockLaterMessages(t *testing.T) {
	a, seed := telegramTest(t)
	if _, err := a.Tx.Exec(a.Context, `DELETE FROM messages WHERE id<>$1`, seed.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Tx.Exec(a.Context, `DELETE FROM messages WHERE id=$1`, seed.ID); err != nil {
		t.Fatal(err)
	}
	attempts := map[string]int{}
	for i := 0; i < 51; i++ {
		writer := APIForFrontend{Context: a.Context, Tx: a.Tx}
		email := fmt.Sprintf("writer-%03d@example.com", i)
		res, err := writer.InsertMessage(generateJwtMessageTemplate(email), APIParamInsertMessage{EmailReceivers: []string{fmt.Sprintf("bad-%03d@example.com", i), fmt.Sprintf("good-%03d@example.com", i)}, MessageContent: "private", InactivePeriodDays: 60, ReminderIntervalDays: 15})
		if err != nil {
			t.Fatal(err)
		}
		saved := res.Data.(MessageData)
		if _, err := a.Tx.Exec(a.Context, `UPDATE messages SET inactive_at=CURRENT_DATE-1,created_at=now()+($2*interval '1 second') WHERE id=$1`, saved.ID, i); err != nil {
			t.Fatal(err)
		}
		if _, err := a.Tx.Exec(a.Context, `INSERT INTO telegram_receivers(message_id,email_receiver,token_hash,chat_id) VALUES($1,$2::text,$2::text,456)`, saved.ID, fmt.Sprintf("good-%03d@example.com", i)); err != nil {
			t.Fatal(err)
		}
	}
	a.SendEmail = func(items []mail.MailItem) []mail.SendEmailsResponse {
		result := make([]mail.SendEmailsResponse, len(items))
		for i, item := range items {
			email := item.To[0].Email
			attempts[email]++
			if strings.HasPrefix(email, "bad-") {
				result[i].Err = errors.New("recipient failure")
			}
		}
		return result
	}
	for i := 0; i < 3; i++ {
		if _, err := a.SendTestamentsOfInactiveMessages(); err != nil {
			t.Fatal(err)
		}
	}
	if attempts["bad-050@example.com"] != 1 || attempts["good-050@example.com"] != 1 {
		t.Fatalf("later recipients were blocked: %v", attempts)
	}
	var queued int
	if err := a.Tx.QueryRow(a.Context, `SELECT count(*) FROM telegram_deliveries WHERE email_receiver='good-050@example.com'`).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Fatal("later Telegram delivery was blocked")
	}
	// Force retry dates due to test the limit without waiting ten hours.
	for i := 0; i < 10; i++ {
		if _, err := a.Tx.Exec(a.Context, `UPDATE email_delivery_receipts SET next_attempt_at=now()`); err != nil {
			t.Fatal(err)
		}
		if _, err := a.SendTestamentsOfInactiveMessages(); err != nil {
			t.Fatal(err)
		}
	}
	for email, count := range attempts {
		want := 1
		if strings.HasPrefix(email, "bad-") {
			want = 10
		}
		if count != want {
			t.Fatalf("%s: got %d attempts, want %d", email, count, want)
		}
	}
	var stopped, advanced int
	if err := a.Tx.QueryRow(a.Context, `SELECT count(*) FROM email_delivery_receipts WHERE stopped_at IS NOT NULL AND sent_at IS NULL`).Scan(&stopped); err != nil {
		t.Fatal(err)
	}
	if err := a.Tx.QueryRow(a.Context, `SELECT count(*) FROM messages WHERE sent_counter=1`).Scan(&advanced); err != nil {
		t.Fatal(err)
	}
	if stopped != 51 || advanced != 51 {
		t.Fatalf("stopped=%d advanced=%d", stopped, advanced)
	}
}

func TestReminderChannelsUseSameBatch(t *testing.T) {
	a, seed := telegramTest(t)
	if _, err := a.Tx.Exec(a.Context, `DELETE FROM messages WHERE id<>$1`, seed.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Tx.Exec(a.Context, `DELETE FROM messages WHERE id=$1`, seed.ID); err != nil {
		t.Fatal(err)
	}
	// Insert in the opposite order from the email schedule.
	for i := 100; i >= 0; i-- {
		email := fmt.Sprintf("linked-%03d@example.com", i)
		writer := APIForFrontend{Context: a.Context, Tx: a.Tx}
		res, err := writer.InsertMessage(generateJwtMessageTemplate(email), APIParamInsertMessage{EmailReceivers: []string{"recipient@example.com"}, MessageContent: "private", InactivePeriodDays: 60, ReminderIntervalDays: 15})
		if err != nil {
			t.Fatal(err)
		}
		saved := res.Data.(MessageData)
		if _, err := a.Tx.Exec(a.Context, `UPDATE messages SET next_reminder_at=CURRENT_DATE-1,created_at=now()+($2*interval '1 second') WHERE id=$1`, saved.ID, i); err != nil {
			t.Fatal(err)
		}
		if _, err := a.Tx.Exec(a.Context, `INSERT INTO telegram_accounts(email,subject,chat_id,phone_hash,reminders_enabled) VALUES($1::text,$1::text,$2,$1::text,true)`, email, 1000+i); err != nil {
			t.Fatal(err)
		}
	}
	a.SendEmail = func(items []mail.MailItem) []mail.SendEmailsResponse {
		return make([]mail.SendEmailsResponse, len(items))
	}
	for run := 0; run < 2; run++ {
		if _, err := a.SendReminderMessages(); err != nil {
			t.Fatal(err)
		}
		var missed int
		if err := a.Tx.QueryRow(a.Context, `SELECT count(*) FROM messages m WHERE next_reminder_at>CURRENT_DATE AND NOT EXISTS(SELECT 1 FROM telegram_deliveries d WHERE d.message_id=m.id AND d.kind='reminder' AND d.cycle_date=CURRENT_DATE-1)`).Scan(&missed); err != nil {
			t.Fatal(err)
		}
		if missed != 0 {
			t.Fatalf("%d advanced reminders have no Telegram queue row", missed)
		}
	}
	var count int
	if err := a.Tx.QueryRow(a.Context, `SELECT count(*) FROM telegram_deliveries WHERE kind='reminder'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 101 {
		t.Fatalf("got %d reminders, want 101", count)
	}
}
