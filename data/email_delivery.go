package data

import (
	"context"
	"github.com/jackc/pgx/v5"
)

// SelectPendingEmails excludes completed work and delayed retries before the limit.
func (q *Queries) SelectPendingEmails(ctx context.Context) ([]SelectInactiveMessagesRow, error) {
	rows, err := q.db.Query(ctx, `-- Select pending recipients before the batch limit.
SELECT
  emails.email AS usr_email,
  emails.created_at AS usr_created_at,
  emails.is_active AS usr_is_active,
  messages.id AS msg_id,
  messages.email_creator AS msg_email_creator,
  messages.created_at AS msg_created_at,
  messages.content_encrypted AS msg_content_encrypted,
  messages.inactive_period_days AS msg_inactive_period_days,
  messages.reminder_interval_days AS msg_reminder_interval_days,
  messages.is_active AS msg_is_active,
  messages.extension_secret AS msg_extension_secret,
  messages.inactive_at AS msg_inactive_at,
  messages.next_reminder_at AS msg_next_reminder_at,
  messages.sent_counter AS msg_sent_counter,
  receivers.message_id AS rcv_message_id,
  receivers.email_receiver AS rcv_email_receiver,
  receivers.is_unsubscribed AS rcv_is_unsubscribed,
  receivers.unsubscribe_secret AS rcv_unsubscribe_secret
FROM
  emails
  INNER JOIN messages ON emails.email = messages.email_creator
  INNER JOIN messages_email_receivers AS receivers ON messages.id = receivers.message_id
WHERE
  messages.inactive_at < CURRENT_DATE
  AND messages.content_encrypted <> ''
  AND messages.is_active
  AND messages.sent_counter < 3
  AND receivers.is_unsubscribed = FALSE
  AND NOT EXISTS (
    SELECT 1 FROM email_delivery_receipts e
    WHERE e.message_id=messages.id AND e.email_receiver=receivers.email_receiver
      AND e.extension_secret=messages.extension_secret AND e.cycle_date=messages.inactive_at
      AND (e.sent_at IS NOT NULL OR e.stopped_at IS NOT NULL OR e.next_attempt_at>now())
  )
ORDER BY
  COALESCE((SELECT e.attempts FROM email_delivery_receipts e
    WHERE e.message_id=messages.id AND e.email_receiver=receivers.email_receiver
      AND e.extension_secret=messages.extension_secret AND e.cycle_date=messages.inactive_at),0),
  messages.created_at ASC,
  messages.id ASC,
  receivers.email_receiver ASC
LIMIT 100
`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[SelectInactiveMessagesRow])
}
