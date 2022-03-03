package api

import (
	"context"
	"testing"

	"github.com/asendia/legacy-api/simple"
	"github.com/google/uuid"
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
	if row.InactiveAt != expectedInactiveAt {
		t.Fatalf("InactiveAt mismatch: %v, expected %v\n", row.InactiveAt, expectedInactiveAt)
	}
}

func TestDeleteMessage(t *testing.T) {
	ctx := context.Background()
	tx, err := pgxPoolConn.Begin(ctx)
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
