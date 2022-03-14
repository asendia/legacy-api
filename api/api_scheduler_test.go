package api

import (
	"context"
	"testing"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/simple"
	"github.com/google/uuid"
)

func TestSelectMessagesNeedReminding(t *testing.T) {
	ctx := context.Background()
	tx, err := pgxPoolConn.Begin(ctx)
	if err != nil {
		t.Errorf("Cannot begin transaction: %v", err)
		return
	}
	defer tx.Rollback(ctx)
	rows := []MessageData{}
	aFe := APIForFrontend{Context: ctx, Tx: tx}
	msgIDsMap := map[uuid.UUID]*MessageData{}
	expectedMessagesEmailReceiversCtr := 0
	for i := 1; i <= 10; i++ {
		msg := generateMessageTemplate()
		expectedMessagesEmailReceiversCtr += len(msg.EmailReceivers)
		res, err := aFe.InsertMessage(
			generateJwtMessageTemplate(msg.EmailCreator),
			APIParamInsertMessage{
				EmailReceivers:       msg.EmailReceivers,
				MessageContent:       msg.MessageContent,
				InactivePeriodDays:   msg.InactivePeriodDays,
				ReminderIntervalDays: msg.ReminderIntervalDays,
			})
		if err != nil {
			t.Errorf("Insert failed: %v", err)
			return
		}
		row := res.Data.(MessageData)
		_, err = tx.Exec(ctx, `UPDATE messages SET next_reminder_at = $1 WHERE id = $2`,
			simple.TimeTodayUTC().Add(-simple.DaysToDuration(i)), row.ID)
		if err != nil {
			t.Error("Failed to update next_reminder_at")
			return
		}
		rows = append(rows, row)
		msgIDsMap[row.ID] = &row
	}
	a := APIForScheduler{Context: ctx, Tx: tx}
	res, err := a.SelectMessagesNeedReminding()
	msgs := res.Data.([]MessageData)
	if err != nil {
		t.Errorf("Select messages need reminding failed: %v", err)
	}
	if len(rows) != 10 || expectedMessagesEmailReceiversCtr != len(msgs) {
		t.Errorf("Messages length inconsistency: %d, expected: %d", len(msgs), expectedMessagesEmailReceiversCtr)
		return
	}
	for _, msg := range msgs {
		if msgIDsMap[msg.ID] == nil || msgIDsMap[msg.ID].InactiveAt != msg.InactiveAt {
			t.Errorf("Inserted msg inconsistent with msg needs reminding")
			return
		}
	}
}

func TestSelectInactiveMessages(t *testing.T) {
	ctx := context.Background()
	tx, err := pgxPoolConn.Begin(ctx)
	if err != nil {
		t.Errorf("Cannot begin transaction: %v", err)
		return
	}
	defer tx.Rollback(ctx)
	aFe := APIForFrontend{Context: ctx, Tx: tx}
	rows := []MessageData{}
	msgIDsMap := map[uuid.UUID]*MessageData{}
	expectedMessagesEmailReceiversCtr := 0
	for i := 1; i <= 10; i++ {
		msg := generateMessageTemplate()
		expectedMessagesEmailReceiversCtr += len(msg.EmailReceivers)
		res, err := aFe.InsertMessage(
			generateJwtMessageTemplate(msg.EmailCreator),
			APIParamInsertMessage{
				EmailReceivers:       msg.EmailReceivers,
				MessageContent:       msg.MessageContent,
				InactivePeriodDays:   msg.InactivePeriodDays,
				ReminderIntervalDays: msg.ReminderIntervalDays,
			})
		if err != nil {
			t.Errorf("Insert failed: %v", err)
			return
		}
		row := res.Data.(MessageData)
		err = tx.QueryRow(ctx, "UPDATE messages SET inactive_at = $1 WHERE id = $2 RETURNING inactive_at;",
			simple.TimeTodayUTC().Add(-simple.DaysToDuration(i)), row.ID).Scan(&row.InactiveAt)
		if err != nil {
			t.Errorf("Failed to update inactive_at")
			return
		}
		rows = append(rows, row)
		msgIDsMap[row.ID] = &row
	}

	a := APIForScheduler{Context: ctx, Tx: tx}
	res, err := a.SelectInactiveMessages()
	msgs := res.Data.([]data.SelectInactiveMessagesRow)
	if err != nil {
		t.Errorf("Select messages need reminding failed: %v", err)
	}
	if len(msgs) != expectedMessagesEmailReceiversCtr {
		t.Errorf("Invalid messages_email_receivers length: %d, expected: %d", len(msgs),
			expectedMessagesEmailReceiversCtr)
		return
	}
	for _, msg := range msgs {
		if msgIDsMap[msg.MsgID] == nil || msgIDsMap[msg.MsgID].InactiveAt != msg.MsgInactiveAt {
			t.Errorf("Inserted msg inconsistent with msg inactive")
			return
		}
	}
}
