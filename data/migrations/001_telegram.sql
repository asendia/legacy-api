-- Apply this file before you enable Telegram. Existing email rows do not change.
BEGIN;
CREATE TABLE telegram_accounts (
  email varchar(70) PRIMARY KEY REFERENCES emails(email) ON DELETE CASCADE,
  subject text NOT NULL UNIQUE,
  chat_id bigint NOT NULL UNIQUE CHECK (chat_id > 0),
  phone_hash text NOT NULL UNIQUE,
  reminders_enabled boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE telegram_login_requests (
  state_hash text PRIMARY KEY,
  proof_hash text NOT NULL,
  verifier text NOT NULL,
  email varchar(70),
  expires_at timestamptz NOT NULL
);
CREATE TABLE telegram_sessions (
  token_hash text PRIMARY KEY,
  email varchar(70) NOT NULL REFERENCES telegram_accounts(email) ON DELETE CASCADE,
  expires_at timestamptz NOT NULL
);
CREATE INDEX telegram_sessions_expiry ON telegram_sessions(expires_at);
CREATE TABLE telegram_receivers (
  message_id uuid NOT NULL,
  email_receiver varchar(70) NOT NULL,
  token_hash text NOT NULL UNIQUE,
  chat_id bigint CHECK (chat_id > 0),
  PRIMARY KEY (message_id, email_receiver),
  FOREIGN KEY (email_receiver, message_id) REFERENCES messages_email_receivers(email_receiver, message_id) ON DELETE CASCADE
);
CREATE TABLE telegram_deliveries (
  id bigserial PRIMARY KEY,
  message_id uuid NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  extension_secret char(69) NOT NULL,
  kind text NOT NULL CHECK (kind IN ('reminder', 'final')),
  cycle_date date NOT NULL,
  email_receiver varchar(70) NOT NULL,
  chat_id bigint NOT NULL CHECK (chat_id > 0),
  body_encrypted text NOT NULL,
  attempts integer NOT NULL DEFAULT 0,
  next_attempt_at timestamptz NOT NULL DEFAULT now(),
  sent_at timestamptz,
  stopped_at timestamptz,
  UNIQUE(message_id, extension_secret, kind, cycle_date, email_receiver)
);
CREATE INDEX telegram_deliveries_due ON telegram_deliveries(next_attempt_at) WHERE sent_at IS NULL AND stopped_at IS NULL;
CREATE TABLE email_delivery_receipts (
  message_id uuid NOT NULL,
  email_receiver varchar(70) NOT NULL,
  extension_secret char(69) NOT NULL,
  cycle_date date NOT NULL,
  sent_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(message_id,email_receiver,extension_secret,cycle_date),
  FOREIGN KEY(email_receiver,message_id) REFERENCES messages_email_receivers(email_receiver,message_id) ON DELETE CASCADE
);
-- Block access through the Supabase public API. The backend uses its database role.
ALTER TABLE telegram_accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE telegram_login_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE telegram_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE telegram_receivers ENABLE ROW LEVEL SECURITY;
ALTER TABLE telegram_deliveries ENABLE ROW LEVEL SECURITY;
ALTER TABLE email_delivery_receipts ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON telegram_accounts, telegram_login_requests, telegram_sessions, telegram_receivers, telegram_deliveries, email_delivery_receipts FROM PUBLIC;
COMMIT;
