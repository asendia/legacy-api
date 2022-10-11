package api

import (
	"context"
	"math"
	"testing"

	"github.com/asendia/legacy-api/data"
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
	res, err := a.InsertMessage(generateJwtMessageTemplate(msg.EmailCreator),
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
	diffDays := row.InactiveAt.Sub(expectedInactiveAt).Hours() / 24
	if math.Abs(diffDays) > 1 {
		t.Fatalf("InactiveAt mismatch: %v, expected %v\n", row.InactiveAt, expectedInactiveAt)
	}
	if len(row.EmailReceivers) != 2 {
		t.Fatal("EmailReceivers length should be 2")
	}
	queries := data.New(tx)
	selectRows, err := queries.SelectMessage(ctx, row.ID)
	if err != nil {
		t.Fatalf("Cannot select by id: %s", row.ID)
	}
	if len(selectRows) != len(row.EmailReceivers) {
		t.Fatalf("Inconsistent length after insertion: %d, should be %d", len(selectRows), len(row.EmailReceivers))
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
		_, err := a.InsertMessage(generateJwtMessageTemplate(msg.EmailCreator),
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
