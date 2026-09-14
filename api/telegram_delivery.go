package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/telegram"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// queueTelegram is also used by queue tests. Send paths pass their selected batch directly.
func (a *APIForScheduler) queueTelegram(kind string) error {
	if os.Getenv("TELEGRAM_ENABLED") != "true" {
		return nil
	}
	queries := data.New(a.Tx)
	if kind == "reminder" {
		rows, err := queries.SelectMessagesNeedReminding(a.Context)
		if err != nil {
			return err
		}
		return a.queueTelegramReminders(rows)
	}
	rows, err := queries.SelectPendingEmails(a.Context)
	if err != nil {
		return err
	}
	return a.queueTelegramFinal(rows)
}

func (a *APIForScheduler) queueTelegramReminders(rows []data.SelectMessagesNeedRemindingRow) error {
	if os.Getenv("TELEGRAM_ENABLED") != "true" {
		return nil
	}
	seen := map[uuid.UUID]bool{}
	for _, row := range rows {
		if seen[row.MsgID] {
			continue
		}
		seen[row.MsgID] = true
		var chatID int64
		err := a.Tx.QueryRow(a.Context, `SELECT t.chat_id FROM telegram_accounts t
            JOIN messages m ON m.email_creator=t.email
            WHERE m.id=$1 AND t.reminders_enabled AND m.inactive_at>=CURRENT_DATE`, row.MsgID).Scan(&chatID)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		body := fmt.Sprintf("Sejiwo reminder. Your message is due after %s. Open Sejiwo to extend the date:\nhttps://sejiwo.com/extend?id=%s&secret=%s", row.MsgInactiveAt.Format("2006-01-02"), row.MsgID, row.MsgExtensionSecret)
		if err := a.insertTelegram(row.MsgID, row.MsgExtensionSecret, "reminder", row.MsgNextReminderAt, row.MsgEmailCreator, chatID, body); err != nil {
			return err
		}
	}
	return nil
}

func (a *APIForScheduler) queueTelegramFinal(rows []data.SelectInactiveMessagesRow) error {
	if os.Getenv("TELEGRAM_ENABLED") != "true" {
		return nil
	}
	for _, row := range rows {
		var chatID int64
		err := a.Tx.QueryRow(a.Context, `SELECT chat_id FROM telegram_receivers WHERE message_id=$1 AND email_receiver=$2 AND chat_id IS NOT NULL`, row.MsgID, row.RcvEmailReceiver).Scan(&chatID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return err
		}
		body, err := DecryptMessageContent(row.MsgContentEncrypted, os.Getenv("ENCRYPTION_KEY"))
		if err != nil {
			return err
		}
		body = "Sejiwo message from " + row.MsgEmailCreator + "\n\n" + body + "\n\nTo stop Telegram delivery, send /stop."
		if isProbablyClientEncrypted(row.MsgContentEncrypted) {
			body += "\nThis text is encrypted. Use CLIENT-AES at https://sejiwo.com with the password from the writer."
		}
		if err := a.insertTelegram(row.MsgID, row.MsgExtensionSecret, "final", row.MsgInactiveAt, row.RcvEmailReceiver, chatID, body); err != nil {
			return err
		}
	}
	return nil
}

func (a *APIForScheduler) insertTelegram(id uuid.UUID, secret, kind string, cycle time.Time, email string, chatID int64, body string) error {
	encrypted, err := telegram.Seal(body, os.Getenv("ENCRYPTION_KEY"))
	if err != nil {
		return err
	}
	_, err = a.Tx.Exec(a.Context, `INSERT INTO telegram_deliveries(message_id,extension_secret,kind,cycle_date,email_receiver,chat_id,body_encrypted) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`, id, secret, kind, cycle, email, chatID, encrypted)
	return err
}

// SendPendingTelegram limits each run. Failed sends remain available for the next run.
func (a *APIForScheduler) SendPendingTelegram() error {
	if os.Getenv("TELEGRAM_ENABLED") != "true" {
		return nil
	}
	ctx := a.Context
	_, err := a.Tx.Exec(ctx, `UPDATE telegram_deliveries d SET stopped_at=now(),body_encrypted='' WHERE d.sent_at IS NULL AND d.stopped_at IS NULL AND (NOT EXISTS (SELECT 1 FROM messages m WHERE m.id=d.message_id AND m.extension_secret=d.extension_secret) OR (d.kind='reminder' AND NOT EXISTS (SELECT 1 FROM telegram_accounts t JOIN messages m ON m.email_creator=t.email WHERE m.id=d.message_id AND t.chat_id=d.chat_id AND t.reminders_enabled AND m.is_active AND m.inactive_at>=CURRENT_DATE)) OR (d.kind='final' AND NOT EXISTS (SELECT 1 FROM telegram_receivers t JOIN messages_email_receivers r ON r.message_id=t.message_id AND r.email_receiver=t.email_receiver WHERE t.message_id=d.message_id AND t.email_receiver=d.email_receiver AND t.chat_id=d.chat_id AND NOT r.is_unsubscribed)))`)
	if err != nil {
		return err
	}
	for i := 0; i < 3; i++ {
		var id, chatID int64
		var encrypted, kind, email string
		var messageID uuid.UUID
		var attempts int
		err := a.Tx.QueryRow(ctx, `SELECT d.id,d.chat_id,d.body_encrypted,d.attempts,d.kind,d.email_receiver,d.message_id FROM telegram_deliveries d JOIN messages m ON m.id=d.message_id AND m.extension_secret=d.extension_secret WHERE d.sent_at IS NULL AND d.stopped_at IS NULL AND d.next_attempt_at<=now() AND (d.kind='final' OR (m.is_active AND m.inactive_at>=CURRENT_DATE)) ORDER BY d.next_attempt_at,d.id LIMIT 1 FOR UPDATE OF d,m SKIP LOCKED`).Scan(&id, &chatID, &encrypted, &attempts, &kind, &email, &messageID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		var currentChat int64
		if kind == "final" {
			err = a.Tx.QueryRow(ctx, `SELECT t.chat_id FROM telegram_receivers t JOIN messages_email_receivers r ON r.message_id=t.message_id AND r.email_receiver=t.email_receiver WHERE t.message_id=$1 AND t.email_receiver=$2 AND t.chat_id=$3 AND NOT r.is_unsubscribed FOR UPDATE OF t,r`, messageID, email, chatID).Scan(&currentChat)
		} else {
			err = a.Tx.QueryRow(ctx, `SELECT chat_id FROM telegram_accounts WHERE email=$1 AND chat_id=$2 AND reminders_enabled FOR UPDATE`, email, chatID).Scan(&currentChat)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			if _, err := a.Tx.Exec(ctx, `UPDATE telegram_deliveries SET stopped_at=now(),body_encrypted='' WHERE id=$1`, id); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		body, err := telegram.Open(encrypted, os.Getenv("ENCRYPTION_KEY"))
		if err != nil {
			return err
		}
		client := telegram.Client{Token: os.Getenv("TELEGRAM_BOT_TOKEN")}
		sendContext, cancel := context.WithTimeout(ctx, 2*time.Second)
		if a.SendTelegram != nil {
			err = a.SendTelegram(sendContext, chatID, body)
		} else {
			err = client.SendText(sendContext, chatID, body)
		}
		cancel()
		if err == nil {
			_, err = a.Tx.Exec(ctx, `UPDATE telegram_deliveries SET sent_at=now(),body_encrypted='' WHERE id=$1`, id)
		} else {
			var sendError *telegram.SendError
			stop := false
			delay := time.Hour
			if errors.As(err, &sendError) {
				stop = sendError.Code == 403 || sendError.Code == 400
				if sendError.RetryAfter > delay {
					delay = sendError.RetryAfter
				}
			}
			if attempts >= 19 {
				stop = true
			}
			_, err = a.Tx.Exec(ctx, `UPDATE telegram_deliveries SET attempts=attempts+1,next_attempt_at=$2,stopped_at=CASE WHEN $3 THEN now() ELSE NULL END,body_encrypted=CASE WHEN $3 THEN '' ELSE body_encrypted END WHERE id=$1`, id, time.Now().Add(delay), stop)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
