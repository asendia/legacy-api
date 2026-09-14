package p

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestUnlinkTelegramAccount(t *testing.T) {
	dsn := os.Getenv("SEJIWO_UNLINK_TEST_URL")
	if dsn == "" {
		t.Skip("Set SEJIWO_UNLINK_TEST_URL to the local sejiwo_unlink_test database with both migrations applied")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if config.Host != "127.0.0.1" || config.Database != "sejiwo_unlink_test" {
		t.Fatal("Use the isolated local unlink test database")
	}
	ctx := context.Background()
	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
 INSERT INTO emails(email) VALUES ('owner@example.com'),('other@example.com'),('recipient@example.com');
 INSERT INTO messages(id,email_creator,content_encrypted,extension_secret,inactive_at,next_reminder_at) VALUES('00000000-0000-0000-0000-000000000001','owner@example.com','saved-content',repeat('s',69),CURRENT_DATE+60,CURRENT_DATE+15);
 INSERT INTO messages_email_receivers(message_id,email_receiver,unsubscribe_secret) VALUES('00000000-0000-0000-0000-000000000001','recipient@example.com',repeat('u',69));
 INSERT INTO telegram_accounts(email,subject,chat_id,phone_hash,reminders_enabled) VALUES('owner@example.com','owner',123,'owner-phone',true),('other@example.com','other',456,'other-phone',true);
 INSERT INTO telegram_sessions(token_hash,email,expires_at) VALUES('owner-session-1','owner@example.com',now()+interval '1 hour'),('owner-session-2','owner@example.com',now()+interval '1 hour'),('other-session','other@example.com',now()+interval '1 hour');
 INSERT INTO telegram_login_requests(state_hash,proof_hash,verifier,email,expires_at) VALUES('owner-state','proof','verifier','owner@example.com',now()+interval '5 minutes'),('other-state','proof','verifier','other@example.com',now()+interval '5 minutes');
 INSERT INTO telegram_receivers(message_id,email_receiver,token_hash,chat_id) VALUES('00000000-0000-0000-0000-000000000001','recipient@example.com','recipient-token',789);
 INSERT INTO telegram_deliveries(message_id,extension_secret,kind,cycle_date,email_receiver,chat_id,body_encrypted) VALUES
 ('00000000-0000-0000-0000-000000000001',repeat('s',69),'reminder',CURRENT_DATE,'owner@example.com',123,'private-reminder'),
 ('00000000-0000-0000-0000-000000000001',repeat('s',69),'reminder',CURRENT_DATE,'other@example.com',456,'other-reminder'),
 ('00000000-0000-0000-0000-000000000001',repeat('s',69),'final',CURRENT_DATE,'recipient@example.com',789,'private-final');`)
	if err != nil {
		t.Fatal(err)
	}
	if err := lockTelegramAccount(ctx, tx, "owner@example.com"); err != nil {
		t.Fatal(err)
	}
	other, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close(ctx)
	for email, want := range map[string]bool{"owner@example.com": false, "other@example.com": true} {
		var acquired bool
		if err := other.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended($1,719283))`, email).Scan(&acquired); err != nil || acquired != want {
			t.Fatalf("account lock mismatch: acquired=%t err=%v", acquired, err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := unlinkTelegramAccount(ctx, tx, "owner@example.com"); err != nil {
			t.Fatal(err)
		}
	}
	for _, query := range []string{
		`SELECT count(*) FROM telegram_accounts WHERE email='owner@example.com'`,
		`SELECT count(*) FROM telegram_sessions WHERE email='owner@example.com'`,
		`SELECT count(*) FROM telegram_login_requests WHERE email='owner@example.com'`,
		`SELECT count(*) FROM telegram_deliveries WHERE email_receiver='owner@example.com' AND (stopped_at IS NULL OR body_encrypted<>'')`,
	} {
		var n int
		if err := tx.QueryRow(ctx, query).Scan(&n); err != nil || n != 0 {
			t.Fatalf("account cleanup failed: count=%d err=%v", n, err)
		}
	}
	for _, query := range []string{
		`SELECT count(*) FROM telegram_accounts WHERE email='other@example.com' AND reminders_enabled`,
		`SELECT count(*) FROM telegram_sessions WHERE email='other@example.com'`,
		`SELECT count(*) FROM telegram_login_requests WHERE email='other@example.com'`,
		`SELECT count(*) FROM telegram_deliveries WHERE kind='final' AND body_encrypted='private-final' AND stopped_at IS NULL`,
		`SELECT count(*) FROM telegram_deliveries WHERE email_receiver='other@example.com' AND body_encrypted='other-reminder' AND stopped_at IS NULL`,
		`SELECT count(*) FROM telegram_receivers WHERE token_hash='recipient-token' AND chat_id=789`,
		`SELECT count(*) FROM messages WHERE content_encrypted='saved-content'`,
		`SELECT count(*) FROM emails WHERE email='owner@example.com' AND is_active`,
		`SELECT count(*) FROM messages_email_receivers WHERE email_receiver='recipient@example.com' AND NOT is_unsubscribed`,
	} {
		var n int
		if err := tx.QueryRow(ctx, query).Scan(&n); err != nil || n != 1 {
			t.Fatalf("unrelated data changed: count=%d err=%v", n, err)
		}
	}
}
