package api

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/asendia/legacy-api/simple"
)

func TestUpdateMessageExtendMessageInactiveAt(t *testing.T) {
	ctx := context.Background()
	tx, err := pgxPoolConn.Begin(ctx)
	if err != nil {
		t.Errorf("Cannot begin transaction: %v", err)
		return
	}
	defer tx.Rollback(ctx)
	a := APIForFrontend{Context: ctx, Tx: tx}
	msg := generateMessageTemplate()
	res, err := a.InsertMessage(
		generateJwtMessageTemplate(msg.EmailCreator),
		APIParamInsertMessage{
			EmailReceivers:       msg.EmailReceivers,
			MessageContent:       msg.MessageContent,
			InactivePeriodDays:   msg.InactivePeriodDays,
			ReminderIntervalDays: msg.ReminderIntervalDays,
		})
	if err != nil {
		t.Errorf("Insert failed: %v\n", err)
		return
	}
	row := res.Data.(MessageData)
	err = tx.QueryRow(ctx, "UPDATE messages SET inactive_at = $1 WHERE id = $2 RETURNING inactive_at;",
		simple.TimeTodayUTC().Add(simple.DaysToDuration(1)), row.ID).Scan(&row.InactiveAt)
	if err != nil {
		t.Errorf("Failed to update inactive_at: %v\n", err)
		return
	}
	res, err = a.ExtendMessageInactiveAt(row.ExtensionSecret, row.ID)
	if err != nil {
		t.Fatalf("ExtendMessage failed: %v", err)
	}
	msgRow := res.Data.(MessageData)
	if msg.InactivePeriodDays != msgRow.InactivePeriodDays {
		t.Fatalf("Data mismatch\n")
	}
	inactiveDaysDiff := math.Abs(msgRow.InactiveAt.Sub(row.InactiveAt).Hours() / 24)
	if inactiveDaysDiff < 10 {
		t.Errorf("InactiveAt not updated: %v expected: %v\n", msgRow.InactiveAt, row.InactiveAt)
	}
	expectedInactiveAt := time.Now().AddDate(0, 0, int(row.InactivePeriodDays))
	inactiveDaysDiff = math.Abs(msgRow.InactiveAt.Sub(expectedInactiveAt).Hours() / 24)
	if row.ID != msgRow.ID || inactiveDaysDiff >= 1 {
		t.Errorf("Unexpected inactiveAt: %v expected: %v\n",
			msgRow.InactiveAt, expectedInactiveAt)
	} else if msgRow.ExtensionSecret == row.ExtensionSecret {
		t.Errorf("Extension secret does not change: %s old: %s", msgRow.ExtensionSecret, row.ExtensionSecret)
	}
}
