package p

import (
	"context"
	"github.com/jackc/pgx/v5"
)

// The caller holds the Telegram delivery lock for the transaction.
func unlinkTelegramAccount(ctx context.Context, tx pgx.Tx, email string) error {
	// Cancel old link requests so a delayed callback cannot restore the account.
	if _, err := tx.Exec(ctx, `DELETE FROM telegram_login_requests WHERE email=$1`, email); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE telegram_deliveries SET stopped_at=now(),body_encrypted='' WHERE kind='reminder' AND email_receiver=$1 AND sent_at IS NULL AND stopped_at IS NULL`, email); err != nil {
		return err
	}
	// The foreign key also deletes all Telegram sessions for this account.
	_, err := tx.Exec(ctx, `DELETE FROM telegram_accounts WHERE email=$1`, email)
	return err
}

// Account locks do not block unrelated login attempts or message delivery.
func lockTelegramAccount(ctx context.Context, tx pgx.Tx, email string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,719283))`, email)
	return err
}
