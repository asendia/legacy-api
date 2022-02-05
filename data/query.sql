-- name: InsertMessage :one
INSERT INTO public.messages (email_creator, email_receivers, message_content, inactive_period_days,
  reminder_interval_days, is_active, extension_secret, inactive_at, next_reminder_at)
  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING
  id, created_at, email_creator, email_receivers, message_content, inactive_period_days,
    reminder_interval_days, is_active, extension_secret, inactive_at, next_reminder_at;

-- name: DeleteMessage :one
DELETE FROM public.messages
WHERE id = $1
RETURNING
  id,
  created_at,
  email_creator,
  email_receivers,
  message_content,
  inactive_period_days,
  reminder_interval_days,
  is_active,
  extension_secret,
  inactive_at,
  next_reminder_at;

-- name: SelectMessagesByEmailCreator :many
SELECT
  id,
  created_at,
  email_creator,
  email_receivers,
  message_content,
  inactive_period_days,
  reminder_interval_days,
  is_active,
  extension_secret,
  inactive_at,
  next_reminder_at
FROM
  public.messages
WHERE
  email_creator = $1;

-- name: SelectMessageByID :one
SELECT
  id,
  created_at,
  email_creator,
  email_receivers,
  message_content,
  inactive_period_days,
  reminder_interval_days,
  is_active,
  extension_secret,
  inactive_at,
  next_reminder_at
FROM
  public.messages
WHERE
  id = $1;

-- name: UpdateMessageExtendsInactiveAt :one
UPDATE
  public.messages
SET
  extension_secret = $1,
  inactive_at = CURRENT_DATE + (inactive_period_days || ' days')::interval,
  next_reminder_at = CURRENT_DATE + (reminder_interval_days || ' days')::interval
WHERE
  id = $2
  AND extension_secret = $3
RETURNING
  id,
  created_at,
  email_creator,
  email_receivers,
  message_content,
  inactive_period_days,
  reminder_interval_days,
  is_active,
  extension_secret,
  inactive_at,
  next_reminder_at;

-- name: UpdateMessage :one
UPDATE
  public.messages
SET
  email_creator = $1,
  email_receivers = $2,
  message_content = $3,
  inactive_period_days = $4,
  reminder_interval_days = $5,
  is_active = $6,
  extension_secret = $7,
  inactive_at = CURRENT_DATE + (inactive_period_days || ' days')::interval,
  next_reminder_at = CURRENT_DATE + (reminder_interval_days || ' days')::interval
WHERE
  id = $8
RETURNING
  id,
  created_at,
  email_creator,
  email_receivers,
  message_content,
  inactive_period_days,
  reminder_interval_days,
  is_active,
  extension_secret,
  inactive_at,
  next_reminder_at;

-- name: SelectMessagesNeedReminding :many
SELECT
  id,
  created_at,
  email_creator,
  email_receivers,
  message_content,
  inactive_period_days,
  reminder_interval_days,
  is_active,
  extension_secret,
  inactive_at,
  next_reminder_at
FROM
  public.messages
WHERE
  next_reminder_at < $1
LIMIT 100;

-- name: UpdateMessageAfterSendingReminder :one
UPDATE
  public.messages
SET
  next_reminder_at = CURRENT_DATE + (reminder_interval_days || ' days')::interval
WHERE
  id = $1
RETURNING
  id,
  created_at,
  email_creator,
  email_receivers,
  message_content,
  inactive_period_days,
  reminder_interval_days,
  is_active,
  extension_secret,
  inactive_at,
  next_reminder_at;

-- name: SelectInactiveMessages :many
SELECT
  id,
  created_at,
  email_creator,
  email_receivers,
  message_content,
  inactive_period_days,
  reminder_interval_days,
  is_active,
  extension_secret,
  inactive_at,
  next_reminder_at
FROM
  public.messages
WHERE
  inactive_at < $1
LIMIT 100;

-- name: UpdateMessageAfterSendingTestament :one
UPDATE
  public.messages
SET
  inactive_at = CURRENT_DATE + (15 || ' days')::interval,
  next_reminder_at = CURRENT_DATE + (30 || ' days')::interval
WHERE
  id = $1
RETURNING
  id,
  created_at,
  email_creator,
  email_receivers,
  message_content,
  inactive_period_days,
  reminder_interval_days,
  is_active,
  extension_secret,
  inactive_at,
  next_reminder_at;
