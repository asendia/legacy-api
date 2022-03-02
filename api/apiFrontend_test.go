package api

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/secure"
	"github.com/asendia/legacy-api/simple"
	"github.com/google/uuid"
)

func TestInsertMessage(t *testing.T) {
	ctx := context.Background()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig())
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
	if row.InactiveAt != expectedInactiveAt {
		t.Fatalf("InactiveAt mismatch: %v, expected %v\n", row.InactiveAt, expectedInactiveAt)
	}
}

func TestSelectMessagesByEmailCreator(t *testing.T) {
	ctx := context.Background()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig())
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
	emailCreator := rdstr + "-inka@kentut.com"
	messageCtr := 0
	rows := []MessageData{}
	a := APIForFrontend{Context: ctx, Tx: tx}
	for i := 0; i < 9; i++ {
		msg := generateMessageTemplate()
		msg.EmailCreator = rdstr + "-" + msg.EmailCreator
		if i%3 == 0 {
			msg.EmailCreator = emailCreator
			messageCtr++
		}
		res, err := a.InsertMessage(generateJwtMessageTemplate(msg.EmailCreator),
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
		rows = append(rows, res.Data.(MessageData))
	}
	res, err := a.SelectMessagesByEmailCreator(generateJwtMessageTemplate(emailCreator))
	if err != nil {
		t.Errorf("SelectMessageByID failed: %v\n", err)
	}
	msgs := res.Data.([]*MessageData)
	if len(msgs) != messageCtr {
		t.Errorf("Inconsistent length: %d, expected: %d\n", len(msgs), len(rows))
	}
}

func TestUpdateMessage(t *testing.T) {
	ctx := context.Background()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig())
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
	rows := []MessageData{}
	a := APIForFrontend{Context: ctx, Tx: tx}
	for i := 0; i < 10; i++ {
		msg := generateMessageTemplate()
		res, err := a.InsertMessage(generateJwtMessageTemplate(msg.EmailCreator),
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
		rows = append(rows, res.Data.(MessageData))
	}
	for id, row := range rows {
		additionalMessage := " UPDATED!!!"
		row.MessageContent += additionalMessage
		// Even = true, odd = false
		row.IsActive = id%2 == 0
		res, err := a.UpdateMessage(generateJwtMessageTemplate(row.EmailCreator),
			APIParamUpdateMessage{
				MessageContent:       row.MessageContent,
				InactivePeriodDays:   row.InactivePeriodDays,
				ReminderIntervalDays: row.ReminderIntervalDays,
				IsActive:             row.IsActive,
				ExtensionSecret:      row.ExtensionSecret,
				ID:                   row.ID,
				EmailReceivers:       row.EmailReceivers,
			})
		if err != nil {
			t.Errorf("Update failed: %v\n", err)
			return
		}
		msg := res.Data.(MessageData)
		if !strings.Contains(msg.MessageContent, additionalMessage) {
			t.Errorf("Message doesn't containt additional string: %s, expected: %s\n", msg.MessageContent, additionalMessage)
		} else if row.IsActive != msg.IsActive {
			t.Error("UpdateMessage failed to change IsActive value\n")
		}
	}
}

func TestDeleteMessage(t *testing.T) {
	ctx := context.Background()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig())
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
	_, err = a.DeleteMessage(generateJwtMessageTemplate("something@notfound"), id)
	if err == nil {
		t.Fatal("DeleteMessage should be failed since there is no items in the table\n")
	}
	msg := generateMessageTemplate()
	res, err := a.InsertMessage(generateJwtMessageTemplate(msg.EmailCreator),
		APIParamInsertMessage{
			EmailReceivers:       msg.EmailReceivers,
			MessageContent:       msg.MessageContent,
			InactivePeriodDays:   msg.InactivePeriodDays,
			ReminderIntervalDays: msg.ReminderIntervalDays,
		})
	if err != nil {
		t.Errorf("InsertMessage failed: %v\n", res)
		return
	}
	row := res.Data.(MessageData)
	_, err = a.DeleteMessage(generateJwtMessageTemplate(row.EmailCreator), row.ID)
	if err != nil {
		t.Errorf("DeleteMessage failed: %v\n", err)
	}
}
func TestUpdateMessageExtendMessageInactiveAt(t *testing.T) {
	ctx := context.Background()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig())
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

func TestValidateEmails(t *testing.T) {
	err := validateEmails([]string{"test@warisin.com", "inavlidemail"})
	if err == nil {
		t.Fatalf("Should detect that some email(s) are invalid")
	}

	err = validateEmails([]string{"test@warisin.com", "test2@waris.in"})
	if err != nil {
		t.Fatalf("validateEmails incorrectly detected valid emails")
	}
}

func TestValidateInactivePeriodDays(t *testing.T) {
	err := validateInactivePeriodDays(30)
	if err == nil {
		t.Fatalf("validateInactivePeriodDays failed to detect invalid inactive period")
	}
	err = validateInactivePeriodDays(90)
	if err != nil {
		t.Fatalf("validateInactivePeriodDays incorrectly detected valid InactivePeriodDays")
	}
}
