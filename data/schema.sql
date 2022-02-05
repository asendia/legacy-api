CREATE TABLE public.messages (
  id uuid NOT NULL DEFAULT gen_random_uuid (),
  created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
  message_content character varying(1000) NOT NULL,
  email_creator character varying(70) NOT NULL,
  email_receivers character varying[] NOT NULL,
  inactive_period_days integer DEFAULT 60 NOT NULL,
  reminder_interval_days integer DEFAULT 15 NOT NULL,
  is_active boolean DEFAULT TRUE NOT NULL,
  extension_secret character (69) NOT NULL,
  inactive_at date NOT NULL,
  next_reminder_at date NOT NULL,
  PRIMARY KEY (id)
);

CREATE INDEX messages_email_creator ON public.messages USING btree (email_creator);

CREATE INDEX messages_next_reminder_at ON public.messages USING btree (next_reminder_at);

CREATE INDEX messages_inactive_at ON public.messages USING btree (inactive_at);

GRANT INSERT, SELECT, UPDATE, DELETE ON public.messages TO project_legacy_admin;
