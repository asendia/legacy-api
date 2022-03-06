CREATE TABLE public.emails (
  email character varying(70) NOT NULL,
  created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
  is_active boolean DEFAULT TRUE NOT NULL,
  PRIMARY KEY (email)
);

CREATE TABLE public.messages (
  id uuid NOT NULL DEFAULT gen_random_uuid (),
  email_creator character varying(70) NOT NULL,
  created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
  content_encrypted character varying(4000) NOT NULL,
  inactive_period_days integer DEFAULT 60 NOT NULL,
  reminder_interval_days integer DEFAULT 15 NOT NULL,
  is_active boolean DEFAULT TRUE NOT NULL,
  extension_secret character (69) NOT NULL,
  inactive_at date NOT NULL,
  next_reminder_at date NOT NULL,
  sent_counter integer DEFAULT 0 NOT NULL,
  PRIMARY KEY (id),
  FOREIGN KEY (email_creator) REFERENCES public.emails (email) ON DELETE CASCADE
);

CREATE TABLE public.messages_email_receivers (
  message_id uuid NOT NULL,
  email_receiver character varying(70) NOT NULL,
  is_unsubscribed boolean DEFAULT FALSE NOT NULL,
  unsubscribe_secret character (69) NOT NULL,
  PRIMARY KEY (email_receiver, message_id),
  FOREIGN KEY (email_receiver) REFERENCES public.emails (email) ON UPDATE CASCADE,
  FOREIGN KEY (message_id) REFERENCES public.messages (id) ON DELETE CASCADE
);

-- For UpdateMessage & DeleteMessage
CREATE INDEX messages_id_email_creator ON public.messages USING btree (id, email_creator);

-- For SelectMessagesNeedReminding
CREATE INDEX messages_need_reminding ON public.messages USING btree (next_reminder_at, is_active);

-- For SelectInactiveMessages
CREATE INDEX messages_select_inactive ON public.messages USING btree (inactive_at, is_active, sent_counter);

-- For SelectMessagesByEmailCreator
CREATE INDEX emails_is_active ON public.emails USING HASH (is_active);

-- For SelectMessagesByEmailCreator, SelectMessagesNeedReminding, SelectInactiveMessages
CREATE INDEX receivers_is_unsubscribed ON public.messages_email_receivers USING HASH (is_unsubscribed);

-- For UpdateMessagesEmailReceiver
CREATE INDEX receivers_id_is_unsubscribed ON public.messages_email_receivers USING btree (message_id,
  unsubscribe_secret);

GRANT INSERT, SELECT, UPDATE, DELETE ON public.emails TO project_legacy_admin;

GRANT INSERT, SELECT, UPDATE, DELETE ON public.messages TO project_legacy_admin;

GRANT INSERT, SELECT, UPDATE, DELETE ON public.messages_email_receivers TO project_legacy_admin;
