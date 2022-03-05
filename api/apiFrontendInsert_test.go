package api

import (
	"context"
	"testing"

	"github.com/asendia/legacy-api/simple"
)

func TestInsertMessage(t *testing.T) {
	ctx := context.Background()
	tx, err := pgxPoolConn.Begin(ctx)
	if err != nil {
		t.Errorf("Cannot begin transaction: %v\n", err)
		return
	}
	defer tx.Rollback(ctx)
	a := APIForFrontend{Context: ctx, Tx: tx}
	msg := generateMessageTemplate()
	res, err := a.InsertMessageV2(generateJwtMessageTemplate(msg.EmailCreator),
		APIParamInsertMessage{
			EmailReceivers:       msg.EmailReceivers,
			MessageContent:       msg.MessageContent,
			InactivePeriodDays:   msg.InactivePeriodDays,
			ReminderIntervalDays: msg.ReminderIntervalDays,
		})
	if err != nil {
		t.Fatalf("InsertMessage failed: %v\n", err)
	}
	row := res.Data.(MessageData)
	if msg.ReminderIntervalDays != row.ReminderIntervalDays || msg.InactivePeriodDays != row.InactivePeriodDays {
		t.Fatalf("Data mismatch: %v, expected %v\n", row, msg)
	}
	expectedInactiveAt := simple.TimeTodayUTC().Add(simple.DaysToDuration(int(msg.InactivePeriodDays)))
	if row.InactiveAt != expectedInactiveAt {
		t.Fatalf("InactiveAt mismatch: %v, expected %v\n", row.InactiveAt, expectedInactiveAt)
	}
}

func BenchmarkInsertMessage(b *testing.B) {
	ctx := context.Background()
	tx, err := pgxPoolConn.Begin(ctx)
	if err != nil {
		b.Fatalf("Failed to connect to DB during BenchInsertMessage: %+v", err)
	}
	defer tx.Rollback(ctx)
	a := APIForFrontend{Context: ctx, Tx: tx}
	for i := 0; i < b.N; i++ {
		msg := generateMessageTemplate()
		_, err := a.InsertMessageV2(generateJwtMessageTemplate(msg.EmailCreator),
			APIParamInsertMessage{
				EmailReceivers:       msg.EmailReceivers,
				MessageContent:       msg.MessageContent,
				InactivePeriodDays:   msg.InactivePeriodDays,
				ReminderIntervalDays: msg.ReminderIntervalDays,
			})
		if err != nil {
			b.Fatalf("InsertMessage failed: %v\n", err)
		}
	}
}

func BenchmarkInsertMessageV2(b *testing.B) {
	ctx := context.Background()
	tx, err := pgxPoolConn.Begin(ctx)
	if err != nil {
		b.Fatalf("Failed to connect to DB during BenchInsertMessage: %+v", err)
	}
	defer tx.Rollback(ctx)
	a := APIForFrontend{Context: ctx, Tx: tx}
	for i := 0; i < b.N; i++ {
		msg := generateMessageTemplate()
		_, err := a.InsertMessageV2(generateJwtMessageTemplate(msg.EmailCreator),
			APIParamInsertMessage{
				EmailReceivers:       msg.EmailReceivers,
				MessageContent:       msg.MessageContent,
				InactivePeriodDays:   msg.InactivePeriodDays,
				ReminderIntervalDays: msg.ReminderIntervalDays,
			})
		if err != nil {
			b.Fatalf("InsertMessage failed: %v\n", err)
		}
	}
}
