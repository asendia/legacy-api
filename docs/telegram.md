# Telegram setup

Telegram is optional and is disabled unless `TELEGRAM_ENABLED=true`. Email stays active. No existing message or recipient row is changed by the migration.

Bot chats do not have end-to-end encryption. Existing CLIENT-AES text stays encrypted during delivery. The writer must give the recipient the password through a separate channel. Normal message text is visible to Telegram. Extension links are bearer secrets: anyone with the link can extend the date.

## Account and recipient flow

1. A writer signs in with the existing Google login and selects **Link Telegram**.
2. Telegram asks for the verified phone number and permission to send bot messages. The backend checks the signed token, issuer, audience, expiry, nonce, and phone claim. Login uses an authorization code, PKCE, and a browser proof that expires after five minutes.
3. The backend links the Telegram subject to the current email account. It does not match accounts by phone number or username. It stores a keyed phone hash, not the phone number. Each phone and Telegram account can link to one email account.
4. The writer can use Telegram to sign in to that same account. Sessions expire after one hour. Google login stays available. Existing users do not have to link Telegram. This is an optional abuse control; it is not proof of a person's identity and does not limit users who keep Google login.
5. The writer can enable extension reminders. This setting is off at first.
6. For a saved recipient, the writer creates a private recipient link. The recipient opens it and presses **Start**. This does not approve the deed. The link contains a random token and no message text. The first private Telegram chat to use it is connected. Treat it like an invitation: share it only with the intended person. The app does not independently verify that person's identity.
7. A new link replaces the old link and removes the prior Telegram connection. **Refresh recipient status** shows whether the recipient has pressed Start. **Remove Telegram delivery** removes the connection. A recipient can send `/stop` to stop all Telegram delivery to their chat. Email unsubscribe stays separate.

The bot cannot find or contact a recipient from a phone number or username alone. Recipients who do nothing continue to receive email.

## BotFather

1. Open [BotFather](https://t.me/BotFather) and send `/newbot`. Use a name that identifies Sejiwo. Save the bot token in Google Cloud Secret Manager as `telegram_bot_token`.
2. In the bot's **Login Widget** settings, add both allowed URLs:
   - `https://sejiwo.com`
   - `https://sejiwo.com/telegram/callback`
3. Keep the default **RS256** signing algorithm. Save the Client Secret as `telegram_client_secret`. Keep the Client ID for the API environment.
4. Generate a separate random webhook secret with at least 32 characters from `A-Z`, `a-z`, `0-9`, `_`, and `-`. Save it as `telegram_webhook_secret`.
5. Set the bot command descriptions: `/start` connects delivery; `/stop` stops Telegram delivery. Do not add the bot to groups. The backend ignores group messages.

Do not put secrets in Git, the frontend, Netlify public variables, or screenshots.

## Deployment order

The checked live service names are `legacy-api` and `legacy-api-scheduler`, in project `monarch-public`, region `asia-southeast1`. The Netlify project is `sejiwo` and uses `asendia/legacy-web`. Production settings were read only during development.

1. Back up the database. Apply `data/migrations/001_telegram.sql`, then `data/migrations/002_email_retries.sql`, once to the production database. If 001 is already applied, apply only 002. Do not run `data/schema.sql`, seed files, or tests against production. Test the migrations on a separate database first.
2. Confirm that the backend database role can read and write the six new tables and use `telegram_deliveries_id_seq`. Row-level security blocks the Supabase public API. A table owner or a role with `BYPASSRLS` can use the tables. If the app uses another database role, give only that role the required table grants and row policies. Do not give access to `anon` or `authenticated`, and do not disable row-level security.
3. Deploy the backend PR with `TELEGRAM_ENABLED` unset or `false` on both services. Check existing Google login, message save, extension, and email delivery.
4. Add these settings. Keep all existing database and Mailjet settings.

| Setting | API service | Scheduler |
| --- | --- | --- |
| `TELEGRAM_ENABLED` | `true` after setup | `true` after setup |
| `TELEGRAM_CLIENT_ID` | BotFather Client ID | Not used |
| `TELEGRAM_CLIENT_SECRET` | Secret Manager reference | Not used |
| `TELEGRAM_BOT_USERNAME` | Bot username without `@` | Not used |
| `TELEGRAM_WEBHOOK_SECRET` | Secret Manager reference | Not used |
| `TELEGRAM_BOT_TOKEN` | Not used for login | Secret Manager reference |
| `ENCRYPTION_KEY` | Existing 32-byte key | Same existing key |

Use additive secret updates. A command with `--set-secrets` can replace existing secret references; keep the database and Mailjet references. Keep the encryption key unchanged during rollout. It is also used to protect queue content and the phone hash. A later key change needs a separate migration plan.

5. Register a Telegram webhook with the Bot API `setWebhook` method. Set `url` to `https://legacy-api-kg4uaex4ca-as.a.run.app/legacy-api-telegram-webhook`, `secret_token` to the webhook secret, and `allowed_updates` to `["message"]`. Keep the bot token out of shell history and logs. Check `getWebhookInfo` for delivery errors. Do not discard pending updates during normal updates.
6. Add a Cloud Scheduler Pub/Sub job for the existing topic `project-legacy-scheduler`, with attribute `action=send-telegram`, every five minutes. Keep the two existing reminder and final-message schedules. This new action only drains stored Telegram deliveries; it does not send email or advance dates. Each run sends at most three Telegram messages, with a two-second timeout per request.
7. Deploy the frontend PR with `PUBLIC_TELEGRAM_ENABLED=false`. After the backend, webhook, and queue job are ready, set it to `true` in Netlify and deploy again. Only this boolean belongs in Netlify. Keep preview deployments disabled unless they use a separate test backend and registered login URL.
8. Use a test writer and consenting test recipient to check login, a reminder, final delivery, `/stop`, a blocked bot, and an email failure. Do not change dates or recipients on a real user's message. A real BotFather configuration is required for this final check.

The existing Cloud Build file reads environment templates. If you use it, preserve the new flags and secret references in your deployment configuration. This change does not enable Telegram or alter production jobs by itself.

## Retry and monitoring

Queue content uses AES-GCM and is removed after success or permanent cancellation. Only hashes of login sessions and recipient invitation tokens are stored. Telegram IDs stay on the backend.

An extension or message edit changes the extension secret and cancels prior pending deliveries. Recipient removal, email unsubscribe, and `/stop` also cancel pending Telegram delivery. The queue checks the current message and recipient state before sending. A request already accepted by Telegram cannot be recalled by a later edit.

Failed requests wait at least one hour, or longer if Telegram asks for it. HTTP 400 and 403 stop that delivery. Other failures stop after 20 attempts. With the five-minute queue job, ready work does not have to wait for the next daily email job. A Telegram failure does not disable an email address. Email success does not discard a pending Telegram delivery.

Telegram does not offer a client idempotency key for `sendMessage`. A timeout or a crash after Telegram accepts a message but before the database commits can cause a duplicate on retry. Database locks prevent concurrent scheduler runs from sending the same pending row. This is at-least-once delivery, not a guarantee of exactly one copy.

With Telegram enabled, final email dates advance once after each recipient succeeds or reaches ten attempts for that cycle. Retries wait one hour. New recipients have priority over retries, and completed or delayed work does not use batch slots. A stopped attempt does not disable the recipient address. The next delivery cycle can try that address again. A crash before receipts commit can still repeat email. Without Telegram enabled, the existing email flow stays available without the new tables; it advances once per message after a successful email.

Monitor `email_delivery_receipts` rows where `stopped_at IS NOT NULL AND sent_at IS NULL`. These are failed deliveries, not successful receipts. Check the provider and the address before you plan another delivery. The `attempts` column records attempts for the current message, recipient, extension secret, and cycle date. Existing successful receipts remain valid after migration 002.

Email and Telegram reminders use the same selected batch. Telegram queue rows are stored before email success can advance the reminder date.

Cycle completion is checked separately from pending sends. If the last pending recipient unsubscribes after a cycle starts, the next scheduler run can complete that cycle without sending a duplicate. Completion requires a receipt for the current extension secret and cycle date, and no subscribed recipient with unfinished work. Up to 100 completed messages advance per run. Later cycles still use the existing 15-day schedule and three-cycle limit.

Check queue counts without reading private content:

```sql
SELECT kind,
       count(*) FILTER (WHERE sent_at IS NULL AND stopped_at IS NULL) AS pending,
       count(*) FILTER (WHERE sent_at IS NULL AND stopped_at IS NOT NULL) AS stopped,
       min(next_attempt_at) FILTER (WHERE sent_at IS NULL AND stopped_at IS NULL) AS oldest_due
FROM telegram_deliveries
GROUP BY kind;
```

Alert on pending work that stays due for more than one hour, new stopped deliveries, and failed scheduler runs. Keep an operator review of these signals. `/stop` and user edits also create stopped rows, so a stopped count alone is not proof of an error.

## Rollback

Set `PUBLIC_TELEGRAM_ENABLED=false` in Netlify and deploy. Set `TELEGRAM_ENABLED=false` on both backend services and pause the new queue job. Existing Google login and email remain available. Keep the added tables so pending work and account links are preserved. Do not delete them during rollback. Existing Telegram sessions stop working while the feature is disabled; users can use Google login.

## Sources

- [Telegram login and signed phone claims](https://core.telegram.org/bots/telegram-login)
- [Bot start requirement](https://core.telegram.org/bots)
- [Bot API webhook and delivery methods](https://core.telegram.org/bots/api)
