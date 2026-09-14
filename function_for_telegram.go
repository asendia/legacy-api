package p

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/asendia/legacy-api/api"
	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/secure"
	"github.com/asendia/legacy-api/telegram"
	"github.com/jackc/pgx/v5"
)

func telegramEnabled() bool { return os.Getenv("TELEGRAM_ENABLED") == "true" }

func telegramLoginClient() telegram.LoginClient {
	return telegram.LoginClient{ClientID: os.Getenv("TELEGRAM_CLIENT_ID"), ClientSecret: os.Getenv("TELEGRAM_CLIENT_SECRET"), RedirectURL: "https://sejiwo.com/telegram/callback"}
}

func telegramTransaction(ctx context.Context, work func(pgx.Tx) error) error {
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig())
	if err != nil {
		return err
	}
	defer conn.Close()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := work(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func verifyTelegramSession(r *http.Request) (secure.JWTResponse, error) {
	var result secure.JWTResponse
	if !telegramEnabled() {
		return result, errors.New("Telegram is disabled")
	}
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !strings.HasPrefix(token, "tg_") || len(token) != 46 {
		return result, errors.New("invalid session")
	}
	err := telegramTransaction(r.Context(), func(tx pgx.Tx) error {
		return tx.QueryRow(r.Context(), `SELECT s.email FROM telegram_sessions s JOIN emails e ON e.email=s.email WHERE token_hash=$1 AND expires_at>now() AND e.is_active`, telegram.Hash(token)).Scan(&result.Email)
	})
	return result, err
}

func TelegramAPI(w http.ResponseWriter, r *http.Request) {
	status, err := api.VerifyCORS(w, r)
	if err != nil || status != http.StatusAccepted {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	if !telegramEnabled() {
		http.Error(w, `{"err":"Telegram is not available"}`, 404)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, `{"err":"Use POST"}`, 405)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	var input struct {
		Action    string `json:"action"`
		State     string `json:"state"`
		Proof     string `json:"proof"`
		Code      string `json:"code"`
		MessageID string `json:"messageId"`
		Email     string `json:"email"`
		Enabled   bool   `json:"enabled"`
	}
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		http.Error(w, `{"err":"Invalid request"}`, 400)
		return
	}
	var user secure.JWTResponse
	if input.Action != "login-start" && input.Action != "login-finish" {
		user, err = VerifyNetlifyJWT(r)
		if err != nil || user.Email == "" {
			http.Error(w, `{"err":"Sign in to use Telegram settings"}`, http.StatusUnauthorized)
			return
		}
	}
	var result interface{}
	err = telegramTransaction(r.Context(), func(tx pgx.Tx) error {
		ctx := r.Context()
		switch input.Action {
		case "login-start", "link-start":
			var email *string
			if input.Action == "link-start" {
				email = &user.Email
			}
			client := telegramLoginClient()
			if client.ClientID == "" || client.ClientSecret == "" {
				return errors.New("Telegram login is not configured")
			}
			state, err := telegram.RandomToken()
			if err != nil {
				return err
			}
			proof, err := telegram.RandomToken()
			if err != nil {
				return err
			}
			verifier, err := telegram.RandomToken()
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(719281)`); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `DELETE FROM telegram_login_requests WHERE expires_at<now()`); err != nil {
				return err
			}
			var count int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM telegram_login_requests`).Scan(&count); err != nil {
				return err
			}
			if count >= 1000 {
				return errors.New("too many login requests; try again later")
			}
			_, err = tx.Exec(ctx, `INSERT INTO telegram_login_requests(state_hash,proof_hash,verifier,email,expires_at) VALUES ($1,$2,$3,$4,now()+interval '5 minutes')`, telegram.Hash(state), telegram.Hash(proof), verifier, email)
			result = map[string]string{"url": client.AuthorizationURL(state, verifier), "state": state, "proof": proof}
			return err
		case "login-finish":
			if len(input.State) != 43 || len(input.Proof) != 43 || input.Code == "" {
				return errors.New("invalid login request")
			}
			var email *string
			var verifier string
			if err := tx.QueryRow(ctx, `DELETE FROM telegram_login_requests WHERE state_hash=$1 AND proof_hash=$2 AND expires_at>now() RETURNING verifier,email`, telegram.Hash(input.State), telegram.Hash(input.Proof)).Scan(&verifier, &email); err != nil {
				return errors.New("login request has expired")
			}
			identity, err := telegramLoginClient().Exchange(ctx, input.Code, verifier, input.State)
			if err != nil {
				return err
			}
			key := os.Getenv("ENCRYPTION_KEY")
			if len(key) != 32 {
				return errors.New("invalid server encryption settings")
			}
			if email != nil {
				if _, err := tx.Exec(ctx, `INSERT INTO emails(email) VALUES ($1) ON CONFLICT DO NOTHING`, *email); err != nil {
					return err
				}
				_, err = tx.Exec(ctx, `INSERT INTO telegram_accounts(email,subject,chat_id,phone_hash) VALUES ($1,$2,$3,$4) ON CONFLICT(email) DO UPDATE SET phone_hash=EXCLUDED.phone_hash WHERE telegram_accounts.subject=EXCLUDED.subject AND telegram_accounts.chat_id=EXCLUDED.chat_id`, *email, identity.Subject, identity.ID, telegram.PhoneHash(identity.Phone, key))
				if err != nil {
					return errors.New("this Telegram account or phone is already linked")
				}
			}
			var accountEmail string
			if err := tx.QueryRow(ctx, `SELECT email FROM telegram_accounts WHERE subject=$1 AND chat_id=$2 AND phone_hash=$3`, identity.Subject, identity.ID, telegram.PhoneHash(identity.Phone, key)).Scan(&accountEmail); err != nil {
				return errors.New("sign in with Google first, then link Telegram in settings")
			}
			if email != nil && accountEmail != *email {
				return errors.New("Telegram is linked to another account")
			}
			token, err := telegram.RandomToken()
			if err != nil {
				return err
			}
			token = "tg_" + token
			if _, err := tx.Exec(ctx, `DELETE FROM telegram_sessions WHERE expires_at<now()`); err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `INSERT INTO telegram_sessions(token_hash,email,expires_at) VALUES ($1,$2,now()+interval '1 hour')`, telegram.Hash(token), accountEmail)
			result = map[string]interface{}{"email": accountEmail, "accessToken": token, "expiresAt": time.Now().Add(time.Hour).UnixMilli()}
			return err
		default:
			if input.Action == "reminders" || input.Action == "receiver-link" || input.Action == "receiver-remove" {
				if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(719282)`); err != nil {
					return err
				}
			}
			switch input.Action {
			case "status":
				var enabled bool
				err := tx.QueryRow(ctx, `SELECT reminders_enabled FROM telegram_accounts WHERE email=$1`, user.Email).Scan(&enabled)
				if err != nil && err != pgx.ErrNoRows {
					return err
				}
				linked := err == nil
				rows, err := tx.Query(ctx, `SELECT t.email_receiver, t.chat_id IS NOT NULL FROM telegram_receivers t JOIN messages m ON m.id=t.message_id WHERE m.id::text=$1 AND m.email_creator=$2`, input.MessageID, user.Email)
				if err != nil {
					return err
				}
				receivers := map[string]bool{}
				for rows.Next() {
					var email string
					var connected bool
					if err := rows.Scan(&email, &connected); err != nil {
						rows.Close()
						return err
					}
					receivers[email] = connected
				}
				err = rows.Err()
				rows.Close()
				if err != nil {
					return err
				}
				result = map[string]interface{}{"linked": linked, "remindersEnabled": enabled, "receivers": receivers}
				return nil
			case "reminders":
				tag, err := tx.Exec(ctx, `UPDATE telegram_accounts SET reminders_enabled=$1 WHERE email=$2`, input.Enabled, user.Email)
				if err != nil {
					return err
				}
				if tag.RowsAffected() != 1 {
					return errors.New("link Telegram first")
				}
			case "receiver-link":
				token, err := telegram.RandomToken()
				if err != nil {
					return err
				}
				tag, err := tx.Exec(ctx, `INSERT INTO telegram_receivers(message_id,email_receiver,token_hash) SELECT r.message_id,r.email_receiver,$1 FROM messages_email_receivers r JOIN messages m ON m.id=r.message_id WHERE m.id::text=$2 AND m.email_creator=$3 AND r.email_receiver=$4 AND NOT r.is_unsubscribed ON CONFLICT(message_id,email_receiver) DO UPDATE SET token_hash=EXCLUDED.token_hash,chat_id=NULL`, telegram.Hash(token), input.MessageID, user.Email, input.Email)
				if err != nil {
					return err
				}
				if tag.RowsAffected() != 1 {
					return errors.New("save this recipient before you create a link")
				}
				result = map[string]string{"url": "https://t.me/" + os.Getenv("TELEGRAM_BOT_USERNAME") + "?start=" + token}
				return nil
			case "receiver-remove":
				_, err := tx.Exec(ctx, `DELETE FROM telegram_receivers r USING messages m WHERE r.message_id=m.id AND m.id::text=$1 AND m.email_creator=$2 AND r.email_receiver=$3`, input.MessageID, user.Email, input.Email)
				return err
			case "logout":
				_, err := tx.Exec(ctx, `DELETE FROM telegram_sessions WHERE token_hash=$1`, telegram.Hash(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")))
				return err
			default:
				return errors.New("invalid Telegram action")
			}
			result = map[string]bool{"ok": true}
			return nil
		}
	})
	if err != nil {
		http.Error(w, `{"err":"Telegram request failed. Check your account settings or try again."}`, http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(result)
}
