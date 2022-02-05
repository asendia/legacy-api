package api

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/secure"
	"github.com/asendia/legacy-api/simple"
	"github.com/google/uuid"
)

func TestInsertMessage(t *testing.T) {
	ctx := context.Background()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig("test"))
	if err != nil {
		t.Fatalf("Cannot connect to DB: %v\n", err)
	}
	defer conn.Close()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Errorf("Cannot begin transaction: %v\n", err)
		return
	}
	defer tx.Rollback(ctx)
	a := APIForFrontend{Context: ctx, Tx: tx}
	msg := generateMessageTemplate()
	res, err := a.InsertMessage(*msg)
	if err != nil {
		t.Fatalf("InsertMessage failed: %v\n", err)
	}
	row := res.Data.(data.InsertMessageRow)
	if msg.ReminderIntervalDays != row.ReminderIntervalDays || msg.InactivePeriodDays != row.InactivePeriodDays {
		t.Fatalf("Data mismatch: %v, expected %v\n", row, msg)
	}
}

func TestSelectMessageByID(t *testing.T) {
	ctx := context.Background()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig("test"))
	if err != nil {
		t.Fatalf("Cannot connect to DB: %v\n", err)
	}
	defer conn.Close()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Errorf("Cannot begin transaction: %v\n", err)
		return
	}
	defer tx.Rollback(ctx)
	rdstr, _ := secure.GenerateRandomString(10)
	emailCreator := rdstr + "inka@kentut.com"
	messageCtr := 0
	rows := []data.InsertMessageRow{}
	for i := 0; i < 10; i++ {
		msg := generateMessageTemplate()
		if i%3 == 0 {
			msg.EmailCreator = emailCreator
			messageCtr++
		}
		row, err := insertTemplateMessage(ctx, tx, msg)
		if err != nil {
			t.Errorf("Insert failed: %v\n", err)
			return
		}
		rows = append(rows, row)
	}
	a := APIForFrontend{Context: ctx, Tx: tx}
	res, err := a.SelectMessageByID(rows[0].ID)
	rMsg := res.Data.(data.SelectMessageByIDRow)
	if err != nil {
		t.Errorf("SelectMessageByID failed: %v\n", err)
	} else if rMsg.EmailCreator != rows[0].EmailCreator ||
		rMsg.InactivePeriodDays != rows[0].InactivePeriodDays {
		t.Errorf("Inconsistent insert: %v, expected: %v\n", rMsg, rows[0])
	}
}

func TestSelectMessagesByEmailCreator(t *testing.T) {
	ctx := context.Background()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig("test"))
	if err != nil {
		t.Fatalf("Cannot connect to DB: %v\n", err)
	}
	defer conn.Close()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Errorf("Cannot begin transaction: %v\n", err)
		return
	}
	defer tx.Rollback(ctx)
	rdstr, _ := secure.GenerateRandomString(10)
	emailCreator := rdstr + "inka@kentut.com"
	messageCtr := 0
	rows := []data.InsertMessageRow{}
	for i := 0; i < 10; i++ {
		msg := generateMessageTemplate()
		msg.EmailCreator = rdstr + msg.EmailCreator
		if i%3 == 0 {
			msg.EmailCreator = emailCreator
			messageCtr++
		}
		row, err := insertTemplateMessage(ctx, tx, msg)
		if err != nil {
			t.Errorf("Insert failed: %v\n", err)
			return
		}
		rows = append(rows, row)
	}
	a := APIForFrontend{Context: ctx, Tx: tx}
	res, err := a.SelectMessagesByEmailCreator(emailCreator)
	if err != nil {
		t.Errorf("SelectMessageByID failed: %v\n", err)
	}
	msgs := res.Data.([]data.SelectMessagesByEmailCreatorRow)
	if len(msgs) != messageCtr {
		t.Errorf("Inconsistent length: %d, expected: %d\n", len(msgs), len(rows))
	} else if msgs[0].EmailCreator != rows[0].EmailCreator || msgs[0].MessageContent != rows[0].MessageContent {
		t.Errorf("Inconsistent insert: %v, expected: %v\n", msgs[0], rows[0])
	}
}

func TestUpdateMessage(t *testing.T) {
	ctx := context.Background()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig("test"))
	if err != nil {
		t.Fatalf("Cannot connect to DB: %v\n", err)
	}
	defer conn.Close()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Errorf("Cannot begin transaction: %v\n", err)
		return
	}
	defer tx.Rollback(ctx)
	rows := []data.InsertMessageRow{}
	for i := 0; i < 10; i++ {
		msg := generateMessageTemplate()
		msg.IsActive = i%2 == 0
		row, err := insertTemplateMessage(ctx, tx, msg)
		if err != nil {
			t.Errorf("Insert failed: %v\n", err)
			return
		}
		rows = append(rows, row)
	}
	a := APIForFrontend{Context: ctx, Tx: tx}
	for id, row := range rows {
		additionalMessage := " UPDATED!!!"
		row.MessageContent += additionalMessage
		row.IsActive = !row.IsActive
		res, err := a.UpdateMessage(data.UpdateMessageParams{
			EmailCreator:         row.EmailCreator,
			EmailReceivers:       row.EmailReceivers,
			MessageContent:       row.MessageContent,
			InactivePeriodDays:   row.InactivePeriodDays,
			ReminderIntervalDays: row.ReminderIntervalDays,
			IsActive:             row.IsActive,
			ID:                   row.ID,
		})
		if err != nil {
			t.Errorf("Update failed: %v\n", err)
			return
		}
		msg := res.Data.(data.UpdateMessageRow)
		msgContent, err := DecryptMessageContent(msg.MessageContent, os.Getenv("ENCRYPTION_KEY"))
		if err != nil {
			t.Errorf("Encryption failed: %v\n", err)
		}
		if ((id%2 == 0) == msg.IsActive) || !strings.Contains(msgContent, additionalMessage) {
			t.Errorf("Updated message is inconsistent: %v, expected: %v\n", msg, row)
			return
		}
	}
}

func TestDeleteMessage(t *testing.T) {
	ctx := context.Background()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig("test"))
	if err != nil {
		t.Fatalf("Cannot connect to DB: %v\n", err)
	}
	defer conn.Close()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Errorf("Cannot begin transaction: %v\n", err)
		return
	}
	defer tx.Rollback(ctx)
	a := APIForFrontend{Context: ctx, Tx: tx}
	id, err := uuid.NewRandom()
	if err != nil {
		t.Fatalf("Failed to generate UUID: %v", err)
	}
	_, err = a.DeleteMessage(id)
	if err == nil {
		t.Fatal("DeleteMessage should be failed since there is no items in the table\n")
	}
	msg := generateMessageTemplate()
	row, err := insertTemplateMessage(ctx, tx, msg)
	if err != nil {
		t.Errorf("InsertMessage failed: %v\n", row)
		return
	}
	_, err = a.DeleteMessage(row.ID)
	if err != nil {
		t.Errorf("DeleteMessage failed: %v\n", err)
	}
}
func TestUpdateMessageExtendMessageInactiveAt(t *testing.T) {
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
	msg := generateMessageTemplate()
	msg.InactiveAt = simple.TimeTodayUTC().Add(simple.DaysToDuration(1))
	row, err := insertTemplateMessage(ctx, tx, msg)
	if err != nil {
		t.Errorf("Insert failed: %v\n", err)
		return
	}
	a := APIForFrontend{Context: ctx, Tx: tx}
	res, err := a.ExtendMessageInactiveAt(row.ID, row.ExtensionSecret)
	if err != nil {
		t.Fatalf("InsertMessage failed: %v", err)
	}
	msgRow := res.Data.(data.UpdateMessageExtendsInactiveAtRow)
	msgRow.MessageContent, err = DecryptMessageContent(msgRow.MessageContent, os.Getenv("ENCRYPTION_KEY"))
	if err != nil {
		t.Fatalf("Decryption failed %v\n", err)
	}
	if msg.MessageContent != msgRow.MessageContent || msg.InactivePeriodDays != msgRow.InactivePeriodDays {
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
