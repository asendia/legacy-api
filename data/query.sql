-- name: InsertMessage :one
WITH insert_email AS (
INSERT INTO emails (email)
    VALUES ($1)
  ON CONFLICT
    DO NOTHING
  RETURNING
    email)
  INSERT INTO messages (email_creator, content_encrypted, inactive_period_days,
    reminder_interval_days, extension_secret, inactive_at, next_reminder_at)
  SELECT
    $1,
    $2,
    $3,
    $4,
    $5,
    CURRENT_DATE + MAKE_INTERVAL(0, 0, 0, $3),
    CURRENT_DATE + MAKE_INTERVAL(0, 0, 0, $4)
  WHERE (
    SELECT
      count(*)
    FROM
      messages
    WHERE
      messages.email_creator = $1) < 3
RETURNING
  *;

-- name: InsertMessageV2 :many
WITH insert_email AS (
INSERT INTO emails (email)
    VALUES ($1)
  ON CONFLICT
    DO NOTHING
  RETURNING
    email
), insert_message AS (
INSERT INTO messages (email_creator, content_encrypted, inactive_period_days,
  reminder_interval_days, extension_secret, inactive_at, next_reminder_at)
  SELECT
    $1,
    $2,
    $3,
    $4,
    $5,
    CURRENT_DATE + MAKE_INTERVAL(0, 0, 0, $3),
    CURRENT_DATE + MAKE_INTERVAL(0, 0, 0, $4)
  WHERE (
    SELECT
      count(*)
    FROM
      messages
    WHERE
      messages.email_creator = $1) < 3
  RETURNING
    *)
INSERT INTO messages_email_receivers
SELECT
  insert_message.id AS message_id,
  unnest(@email_receivers::text[]) AS email_receiver,
  unnest(@unsubscribe_secrets::text[]) AS unsubscribe_secret
RETURNING
  *;

-- name: InsertMessagesEmailReceivers :many
WITH insert_email AS (
INSERT INTO emails
  SELECT
    unnest(@email_receivers::text[]) AS email
  ON CONFLICT
    DO NOTHING)
INSERT INTO messages_email_receivers (message_id, email_receiver, unsubscribe_secret)
SELECT
  unnest(@message_ids::uuid[]) AS message_id,
  unnest(@email_receivers::text[]) AS email_receiver,
  unnest(@unsubscribe_secrets::text[]) AS unsubscribe_secret
RETURNING
  *;

-- name: UpdateMessagesEmailReceiverUnsubscribe :one
UPDATE
  messages_email_receivers
SET
  is_unsubscribed = TRUE
WHERE
  message_id = $1
  AND unsubscribe_secret = $2
RETURNING
  *;

-- name: DeleteMessage :one
DELETE FROM messages
WHERE id = $1
  AND email_creator = $2
RETURNING
  *;

-- name: SelectMessagesByEmailCreator :many
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
  INNER JOIN messages ON messages.email_creator = emails.email
  LEFT JOIN messages_email_receivers AS receivers ON messages.id = receivers.message_id
WHERE
  messages.email_creator = $1
  AND emails.is_active
ORDER BY
  messages.created_at ASC
LIMIT 30;

-- name: UpdateMessageExtendsInactiveAt :one
UPDATE
  messages
SET
  extension_secret = $1,
  inactive_at = CURRENT_DATE + MAKE_INTERVAL(0, 0, 0, inactive_period_days),
  next_reminder_at = CURRENT_DATE + MAKE_INTERVAL(0, 0, 0, reminder_interval_days)
WHERE
  id = $2
  AND extension_secret = $3
  AND inactive_at >= CURRENT_DATE
  AND is_active
RETURNING
  *;

-- name: UpdateEmail :exec
UPDATE
  emails
SET
  is_active = $1
WHERE
  email = $2;

-- name: UpdateMessage :one
WITH insert_emails AS (
INSERT INTO emails
  SELECT
    unnest(@email_receivers::text[]) AS email
  ON CONFLICT (email)
    DO NOTHING
),
delete_receivers AS (
  DELETE FROM messages_email_receivers
  WHERE message_id = $6
    AND is_unsubscribed = FALSE
),
insert_receivers AS (
INSERT INTO messages_email_receivers
  SELECT
    unnest(@message_ids::uuid[]) AS message_id,
    unnest(@email_receivers::text[]) AS email_receiver,
    unnest(@is_unsubscribeds::boolean[]) AS is_unsubscribed,
    unnest(@unsubscribe_secrets::text[]) AS unsubscribe_secret
  ON CONFLICT
    DO NOTHING)
UPDATE
  messages
SET
  content_encrypted = $1,
  inactive_period_days = $2,
  reminder_interval_days = $3,
  is_active = $4,
  extension_secret = $5,
  inactive_at = CURRENT_DATE + MAKE_INTERVAL(0, 0, 0, $2),
  next_reminder_at = CURRENT_DATE + MAKE_INTERVAL(0, 0, 0, $3),
  sent_counter = 0
WHERE
  id = $6
  AND email_creator = $7
RETURNING
  *;

-- name: SelectMessagesNeedReminding :many
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
  message_id AS rcv_message_id,
  receivers.email_receiver AS rcv_email_receiver,
  receivers.is_unsubscribed AS rcv_is_unsubscribed,
  receivers.unsubscribe_secret AS rcv_unsubscribe_secret
FROM
  emails
  INNER JOIN messages ON emails.email = messages.email_creator
  INNER JOIN messages_email_receivers AS receivers ON messages.id = receivers.message_id
WHERE
  messages.is_active
  AND messages.content_encrypted <> ''
  AND messages.next_reminder_at <= CURRENT_DATE
  AND receivers.is_unsubscribed = FALSE
ORDER BY
  messages.created_at ASC,
  messages.id ASC
LIMIT 100;

-- name: UpdateMessageAfterSendingReminder :one
UPDATE
  messages
SET
  next_reminder_at = CURRENT_DATE + MAKE_INTERVAL(0, 0, 0, reminder_interval_days)
WHERE
  id = $1
RETURNING
  *;

-- name: SelectInactiveMessages :many
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
ORDER BY
  messages.created_at ASC,
  messages.id ASC
LIMIT 100;

-- name: UpdateMessageAfterSendingTestament :one
UPDATE
  messages
SET
  is_active = CASE WHEN sent_counter < 2 THEN
    is_active
  ELSE
    FALSE
  END,
  sent_counter = sent_counter + 1,
  inactive_at = CURRENT_DATE + MAKE_INTERVAL(0, 0, 0, 15),
  next_reminder_at = CURRENT_DATE + MAKE_INTERVAL(0, 0, 0, 30)
WHERE
  id = $1
  AND sent_counter < 3
  AND is_active
RETURNING
  *;
