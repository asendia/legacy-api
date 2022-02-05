package api

import (
	"context"
	"math"
	"testing"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/simple"
	"github.com/google/uuid"
)

func TestSelectMessagesNeedReminding(t *testing.T) {
	ctx := context.Background()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig("test"))
	if err != nil {
		t.Fatalf("Cannot connect to DB: %v", err)
	}
	defer conn.Close()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Errorf("Cannot begin transaction: %v", err)
		return
	}
	defer tx.Rollback(ctx)
	rows := []data.InsertMessageRow{}
	msgIDsMap := map[uuid.UUID]*data.InsertMessageRow{}
	for i := 1; i <= 10; i++ {
		msg := generateMessageTemplate()
		msg.NextReminderAt = simple.TimeTodayUTC().Add(-simple.DaysToDuration(i))
		row, err := insertTemplateMessage(ctx, tx, msg)
		if err != nil {
			t.Errorf("Insert failed: %v", err)
			return
		}
		rows = append(rows, row)
		msgIDsMap[row.ID] = &row
	}

	a := APIForScheduler{Context: ctx, Tx: tx}
	res, err := a.SelectMessagesNeedReminding()
	msgs := res.Data.([]data.SelectMessagesNeedRemindingRow)
	if err != nil {
		t.Errorf("Select messages need reminding failed: %v", err)
	}
	if len(rows) != 10 || len(rows) != len(msgs) {
		t.Errorf("Length inconsistency, inserted msgs inconsistent with msgs needs reminding")
		return
	}
	for _, msg := range msgs {
		if msgIDsMap[msg.ID] == nil || msgIDsMap[msg.ID].InactiveAt != msg.InactiveAt {
			t.Errorf("Inserted msg inconsistent with msg needs reminding")
			return
		}
	}
}

func TestUpdateMessageAfterSendingReminder(t *testing.T) {
	ctx := context.Background()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig("test"))
	if err != nil {
		t.Fatalf("Cannot connect to DB: %v", err)
	}
	defer conn.Close()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Errorf("Cannot begin transaction: %v", err)
		return
	}
	defer tx.Rollback(ctx)
	today := simple.TimeTodayUTC()
	msgT := generateMessageTemplate()
	msgT.NextReminderAt = today
	row, err := insertTemplateMessage(ctx, tx, msgT)
	if err != nil {
		t.Errorf("Insert failed: %v", err)
		return
	}

	a := APIForScheduler{Context: ctx, Tx: tx}
	reminderDiffDays := simple.DurationToDays(row.NextReminderAt.Sub(today))
	if math.Round(reminderDiffDays) != 0 {
		t.Errorf("Wrong nextReminderAt inserted")
		return
	}
	res, err := a.UpdateMessageAfterSendingReminder(row.ID)
	if err != nil {
		t.Errorf("Update message failed: %v", err)
		return
	}
	msg := res.Data.(data.UpdateMessageAfterSendingReminderRow)
	reminderDiffDays = simple.DurationToDays(msg.NextReminderAt.Sub(today))
	if int32(math.Round(reminderDiffDays)) != msg.ReminderIntervalDays {
		t.Errorf("Incorrect update on nextReminderAt: %f expected: %d", reminderDiffDays, msg.ReminderIntervalDays)
	}
}

func TestSelectInactiveMessages(t *testing.T) {
	ctx := context.Background()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig("test"))
	if err != nil {
		t.Fatalf("Cannot connect to DB: %v", err)
	}
	defer conn.Close()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Errorf("Cannot begin transaction: %v", err)
		return
	}
	defer tx.Rollback(ctx)
	rows := []data.InsertMessageRow{}
	msgIDsMap := map[uuid.UUID]*data.InsertMessageRow{}
	for i := 1; i <= 10; i++ {
		msgT := generateMessageTemplate()
		msgT.InactiveAt = simple.TimeTodayUTC().Add(-simple.DaysToDuration(i))
		row, err := insertTemplateMessage(ctx, tx, msgT)
		if err != nil {
			t.Errorf("Insert failed: %v", err)
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
	if len(msgs) != len(rows) {
		t.Errorf("Length inconsistency, inserted msgs inconsistent with msgs inactive")
		return
	}
	for _, msg := range msgs {
		if msgIDsMap[msg.ID] == nil || msgIDsMap[msg.ID].InactiveAt != msg.InactiveAt {
			t.Errorf("Inserted msg inconsistent with msg inactive")
			return
		}
	}
}

func TestUpdateMessageAfterSendingTestament(t *testing.T) {
	ctx := context.Background()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig("test"))
	if err != nil {
		t.Fatalf("Cannot connect to DB: %v", err)
	}
	defer conn.Close()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Errorf("Cannot begin transaction: %v", err)
		return
	}
	defer tx.Rollback(ctx)
	today := simple.TimeTodayUTC()
	msgT := generateMessageTemplate()
	msgT.InactiveAt = today
	row, err := insertTemplateMessage(ctx, tx, msgT)
	if err != nil {
		t.Errorf("Insert failed: %v", err)
		return
	}

	a := APIForScheduler{Context: ctx, Tx: tx}
	inactiveDiffNsec := float64(row.InactiveAt.Sub(today) / 24 / 3600)
	inactiveDiffDays := math.Round(inactiveDiffNsec / 1000000000)
	if inactiveDiffDays != 0 {
		t.Errorf("Wrong inactiveAt inserted")
		return
	}
	res, err := a.UpdateMessageAfterSendingTestament(row.ID)
	if err != nil {
		t.Errorf("Update message failed: %v", err)
		return
	}
	msg := res.Data.(data.UpdateMessageAfterSendingTestamentRow)
	inactiveDiffNsec = float64(msg.InactiveAt.Sub(today) / 24 / 3600)
	inactiveDiffDays = inactiveDiffNsec / 1000000000
	inactiveDiffAfterTestamentIsSentConst := 15
	if int(math.Round(inactiveDiffDays)) != inactiveDiffAfterTestamentIsSentConst {
		t.Errorf("Incorrect update on inactiveAt: %f expected: %d", inactiveDiffDays, inactiveDiffAfterTestamentIsSentConst)
	}
}
