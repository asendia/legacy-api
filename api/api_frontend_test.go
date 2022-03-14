package api

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

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
	err := validateEmails([]string{"test@sejiwo.com", "inavlidemail"})
	if err == nil {
		t.Fatalf("Should detect that some email(s) are invalid")
	}
	err = validateEmails([]string{"test@sejiwo.com", "test2@waris.in"})
	if err != nil {
		t.Fatalf("validateEmails incorrectly detected valid emails")
	}
	err = validateEmails([]string{})
	if err != nil {
		t.Fatalf("Empty email list is valid in this service")
	}
}

func TestValidateInactivePeriodDays(t *testing.T) {
	err := validateInactivePeriodDays(29)
	if err == nil {
		t.Fatalf("validateInactivePeriodDays failed to detect invalid inactive period")
	}
	err = validateInactivePeriodDays(90)
	if err != nil {
		t.Fatalf("validateInactivePeriodDays incorrectly detected valid InactivePeriodDays")
	}
	err = validateInactivePeriodDays(30)
	if err != nil {
		t.Fatalf("validateInactivePeriodDays incorrectly detected valid InactivePeriodDays")
	}
}
