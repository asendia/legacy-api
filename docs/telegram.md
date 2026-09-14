# Telegram setup

Telegram is optional. The production templates set `TELEGRAM_ENABLED=true` to preserve the configured production service. Other environments are disabled unless `TELEGRAM_ENABLED=true`. Email stays active. No existing message or recipient row is changed by the migration.

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
2. Open the [BotFather mini app](https://t.me/botfather?startapp=), select the bot, then **Login Widget**. If the page shows a single domain field, select **Switch to OpenID Connect Login**. Under **Redirect URIs**, add exactly `https://sejiwo.com/telegram/callback`. Leave **Trusted Origins** and **Native Login** empty for this server-side token exchange.
3. Keep **RS256**. Save the Client Secret as `telegram_client_secret`. The public Client ID is `8927237838`; the bot username is `SejiwoBot`.
4. Generate a separate 64-character webhook secret using random letters and numbers. Uppercase and lowercase are valid. Save it as `telegram_webhook_secret`. The backend requires at least 32 characters; Telegram accepts up to 256 characters from `A-Z`, `a-z`, `0-9`, `_`, and `-`.
5. Set the bot command descriptions: `/start` connects delivery; `/stop` stops Telegram delivery. Do not add the bot to groups. The backend ignores group messages.

Do not put secrets in Git, the frontend, Netlify public variables, or screenshots.

## Deployment order

The checked live service names are `legacy-api` and `legacy-api-scheduler`, in project `monarch-public`, region `asia-southeast1`. The Netlify project is `sejiwo` and uses `asendia/legacy-web`. The operator completed backend setup through the Google Cloud dashboard. Secret values are not stored in this repository.

1. Back up the database. Apply `data/migrations/001_telegram.sql`, then `data/migrations/002_email_retries.sql`, once to the production database. If 001 is already applied, apply only 002. Do not run `data/schema.sql`, seed files, or tests against production. Test the migrations on a separate database first.
2. Confirm that the backend database role can read and write the six new tables and use `telegram_deliveries_id_seq`. Row-level security blocks the Supabase public API. A table owner or a role with `BYPASSRLS` can use the tables. If the app uses another database role, give only that role the required table grants and row policies. Do not give access to `anon` or `authenticated`, and do not disable row-level security.
3. For a new environment, set `TELEGRAM_ENABLED=false` in both deployment templates before the first deployment. Check existing Google login, message save, extension, and email delivery.
4. Add these settings. Keep all existing database and Mailjet settings.

| Setting | API service | Scheduler |
| --- | --- | --- |
| `TELEGRAM_ENABLED` | `true` after setup | `true` after setup |
| `TELEGRAM_CLIENT_ID` | `8927237838` | Not used |
| `TELEGRAM_CLIENT_SECRET` | Secret Manager reference | Not used |
| `TELEGRAM_BOT_USERNAME` | `SejiwoBot` | Not used |
| `TELEGRAM_WEBHOOK_SECRET` | Secret Manager reference | Not used |
| `TELEGRAM_BOT_TOKEN` | Not used for login | Secret Manager reference |
| `ENCRYPTION_KEY` | Existing 32-byte key | Same existing key |

`cloudbuild.yaml` uses `--update-secrets` with `telegram_client_secret:latest` and `telegram_webhook_secret:latest` on the API, and `telegram_bot_token:latest` on the scheduler. Use additive secret updates. A command with `--set-secrets` replaces existing secret references; keep the database and Mailjet references. Keep the encryption key unchanged during rollout. It is also used to protect queue content and the phone hash. A later key change needs a separate migration plan.

5. Register a Telegram webhook with the Bot API `setWebhook` method. Set `url` to `https://legacy-api-kg4uaex4ca-as.a.run.app/legacy-api-telegram-webhook`, `secret_token` to the webhook secret, and `allowed_updates` to `["message"]`. Keep the bot token out of shell history and logs. Check `getWebhookInfo` for delivery errors. Do not discard pending updates during normal updates.
6. Use the existing `SendTelegramMessages` Cloud Scheduler Pub/Sub job described below. For a new environment, create it once. Keep the two existing reminder and final-message schedules. This new action only drains stored Telegram deliveries; it does not send email or advance dates. Each run sends at most three Telegram messages, with a two-second timeout per request.
7. The frontend `netlify.toml` sets `PUBLIC_TELEGRAM_ENABLED=true` for production and `false` for other deploy contexts. The flag is read at build time. For a new environment, keep it false until the backend, webhook, and queue job are ready, then build and deploy again. Only this public boolean belongs in Netlify. Keep preview deployments disabled unless they use a separate test backend and registered login URL.
8. Use a test writer and consenting test recipient to check login, a reminder, final delivery, `/stop`, a blocked bot, and an email failure. Do not change dates or recipients on a real user's message. A real BotFather configuration is required for this final check.

## Production schedules

All jobs use region `asia-southeast1`, time zone `Asia/Jakarta`, and Pub/Sub topic `project-legacy-scheduler`.

| Job | Frequency | Message attribute `action` |
| --- | --- | --- |
| `SendReminderMessages` | `22 19 * * *` | `send-reminder-messages` |
| `SendTestaments` | `38 19 * * *` | `send-testaments` |
| `SendTelegramMessages` | `48 19 * * *` | `send-telegram` |

The Telegram job body is `{}`. Put `action` in message attributes, not in the body. The daily schedule is an operator cost preference. The first two jobs also attempt queued Telegram deliveries after their normal work. Each invocation attempts at most three Telegram messages. A large queue can take several days to clear. Job names are labels; the code dispatches by the `action` attribute. Do not create duplicate jobs.

## Configuration ownership

Change production environment values in the templates and secret references in `cloudbuild.yaml`. Keep actual secrets in Google Cloud Secret Manager. Production secret references use `latest` by operator choice. New container instances resolve that version; redeploy affected services after a secret change so all instances use the intended value. If the webhook secret changes, update the Telegram webhook registration to match.

Dashboard changes can be used for an urgent rollback, but update the repository before the next automated deployment. `--env-vars-file` replaces environment variables. The existing scheduler jobs, BotFather settings, webhook registration, and secret values remain external resources; this pipeline does not create or modify them.

The frontend configuration lives in `legacy-web/netlify.toml`. Its production setting takes precedence over a duplicate Netlify dashboard build setting. Remove duplicate dashboard flags after the repository deployment is verified. A code or config deployment is not proof that live Telegram login and final delivery have passed.

## Register and check the webhook locally

Use a local terminal if preferred; Cloud Shell is not required. The script reads secrets without echo and passes them to curl through standard input. It sends the bot token and webhook secret only to Telegram. Do not use the Client Secret in place of the bot token. Keep shell tracing disabled.

```bash
bash <<'BASH'
LC_ALL=C
read -r -s -p "Bot token: " bot_token </dev/tty
printf '\n'
read -r -s -p "Webhook secret: " webhook_secret </dev/tty
printf '\n'
if [[ ! "$bot_token" =~ ^8927237838:[A-Za-z0-9_-]+$ ]]; then
  echo "Invalid bot token or wrong bot ID."
  exit 1
fi
if [[ ! "$webhook_secret" =~ ^[A-Za-z0-9_-]+$ ]] ||
   (( ${#webhook_secret} < 32 || ${#webhook_secret} > 256 )); then
  echo "Invalid webhook secret characters or length."
  exit 1
fi
{
  printf 'url = "https://api.telegram.org/bot%s/setWebhook"\n' "$bot_token"
  printf 'data-urlencode = "secret_token=%s"\n' "$webhook_secret"
} | curl --config - --silent --show-error \
  --connect-timeout 10 --max-time 30 \
  --data-urlencode 'url=https://legacy-api-kg4uaex4ca-as.a.run.app/legacy-api-telegram-webhook' \
  --data-urlencode 'allowed_updates=["message"]' \
  --data-urlencode 'drop_pending_updates=false'
printf '\n'
printf 'url = "https://api.telegram.org/bot%s/getWebhookInfo"\n' "$bot_token" |
  curl --config - --silent --show-error --connect-timeout 10 --max-time 30
printf '\n'
unset bot_token webhook_secret
BASH
```

Check the JSON result for `ok: true`, the expected webhook URL, pending updates, and any delivery error. HTTP success alone is not sufficient. Send `/start` from a test account after the API is enabled, then check webhook status and API logs. A plain `/start` has no bot reply. An empty pending count alone is not proof of a complete login or delivery test.

The character check and length check are separate because the macOS regular expression engine rejects the interval `{32,256}`. Do not replace a valid secret because of that invalid pattern.

## Retry and monitoring

Queue content uses AES-GCM and is removed after success or permanent cancellation. Only hashes of login sessions and recipient invitation tokens are stored. Telegram IDs stay on the backend.

An extension or message edit changes the extension secret and cancels prior pending deliveries. Recipient removal, email unsubscribe, and `/stop` also cancel pending Telegram delivery. The queue checks the current message and recipient state before sending. A request already accepted by Telegram cannot be recalled by a later edit.

Failed requests wait at least one hour, or longer if Telegram asks for it. HTTP 400 and 403 stop that delivery. Other failures stop after 20 attempts. With the selected daily queue job, retries can wait until a later daily run. A Telegram failure does not disable an email address. Email success does not discard a pending Telegram delivery.

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

With these daily schedules, alert on pending work that stays due for more than 24 hours, new stopped deliveries, and failed scheduler runs. Review queue size against the small daily send limit. Keep an operator review of these signals. `/stop` and user edits also create stopped rows, so a stopped count alone is not proof of an error.

## Rollback

Set `PUBLIC_TELEGRAM_ENABLED=false` in the production context of the frontend `netlify.toml` and deploy. Set `TELEGRAM_ENABLED=false` in both backend production templates and deploy, or use an urgent dashboard change followed by the same repository change and pause the new queue job. Existing Google login and email remain available. Keep the added tables so pending work and account links are preserved. Do not delete them during rollback. Existing Telegram sessions stop working while the feature is disabled; users can use Google login.

## Sources

- [Telegram login and signed phone claims](https://core.telegram.org/bots/telegram-login)
- [Bot start requirement](https://core.telegram.org/bots)
- [Bot API webhook and delivery methods](https://core.telegram.org/bots/api)

## Login failure codes

The callback can fail after Telegram approves the request. The API returns a fixed `code` for the failed step and writes that code to the service log as `Telegram login failed`. It does not log credentials, token claims, phone numbers, or raw provider errors.

- `telegram_client_settings`: check that the API uses the Client Secret from BotFather, not the bot token. Check the active Secret Manager version and redeploy after a change.
- `telegram_code_rejected` or `login_expired`: start a new login in the same browser tab. Check that the callback URL matches exactly.
- `telegram_phone_required`: allow Telegram to share the verified phone number.
- `telegram_link_required`: sign in with Google, then link Telegram in Account settings.
- `telegram_link_conflict`: the account cannot be linked to the selected Sejiwo account.
- `telegram_token_verification`, `telegram_nonce`, `telegram_token_time`, or `telegram_identity`: inspect the provider configuration and validation step. Keep signature, issuer, audience, nonce, time, and phone checks enabled.
- `login_storage`: check database access, migrations, and service health.
- `login_server_settings`: check the existing encryption key configuration without changing the key.

These codes identify the failed step. They do not prove that a live login fault is fixed. After a diagnostic deployment, retry once and use the safe code to select the next check. Do not collect or publish tokens or callback query strings.

### Account ID format

The signed `id` claim can contain an integer or a decimal integer string. Both forms are read as the same `int64` value, without conversion through floating point. The subject remains separate and must be present. Do not use the subject as a fallback chat ID: the account key and delivery address must not change because a claim is absent.

Missing, null, zero, negative, fractional, and out-of-range IDs are rejected. Token validation and verified-phone checks still apply. Identity failures log only a fixed reason (`claim_encoding`, `invalid_or_missing_id`, or `missing_subject`), never claim values.

The numeric/string ID variation is also documented in the [Telegram OIDC library claim type](https://pkg.go.dev/github.com/tergeoo/telegram-go/oidc#OIDCClaims). The official [Telegram claim example](https://core.telegram.org/bots/telegram-login#user-data-structure) shows the numeric form. No extra library is required for this parsing step.
