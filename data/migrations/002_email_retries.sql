-- Apply after 001_telegram.sql and before you enable Telegram.
BEGIN;
ALTER TABLE email_delivery_receipts ALTER COLUMN sent_at DROP NOT NULL;
ALTER TABLE email_delivery_receipts ADD COLUMN attempts integer NOT NULL DEFAULT 0;
ALTER TABLE email_delivery_receipts ADD COLUMN next_attempt_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE email_delivery_receipts ADD COLUMN stopped_at timestamptz;
COMMIT;
