package api

import (
	"context"
	"math"
	"testing"

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
	msg.InactiveAt = simple.TimeTodayUTC().Add(simple.DaysToDuration(1))
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
	res, err = a.ExtendMessageInactiveAt(row.ExtensionSecret, row.ID)
	if err != nil {
		t.Fatalf("ExtendMessage failed: %v", err)
	}
	msgRow := res.Data.(MessageData)
	if msg.InactivePeriodDays != msgRow.InactivePeriodDays {
		t.Fatalf("Data mismatch\n")
	}
	inactivePeriodDiffNsec := float64(msgRow.InactiveAt.Sub(simple.TimeTodayUTC()) / 24 / 3600)
	inactivePeriodDiffDays := math.Round(inactivePeriodDiffNsec / 1000000000)
	if row.ID != msgRow.ID || int32(inactivePeriodDiffDays) != msgRow.InactivePeriodDays {
		t.Errorf("Inactive period is not updated: %v expected: %v\n",
			msgRow.InactiveAt, row.InactiveAt.Add(simple.DaysToDuration(int(row.InactivePeriodDays))))
	} else if msgRow.ExtensionSecret == row.ExtensionSecret {
		t.Errorf("Extension secret does not change: %s old: %s", msgRow.ExtensionSecret, row.ExtensionSecret)
	}
}
