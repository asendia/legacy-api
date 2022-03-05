package api

import (
	"context"
	"testing"

	"github.com/asendia/legacy-api/secure"
)

func TestSelectMessagesByEmailCreator(t *testing.T) {
	ctx := context.Background()
	tx, err := pgxPoolConn.Begin(ctx)
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
	for i := 0; i < 5; i++ {
		msg := generateMessageTemplate()
		msg.EmailCreator = rdstr + "-" + msg.EmailCreator

		if i%2 == 0 {
			msg.EmailCreator = emailCreator
			messageCtr++
		}
		if i == 2 {
			msg.EmailReceivers = []string{}
		} else if i == 4 {
			msg.MessageContent = ""
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
		t.Fatalf("Inconsistent length: %d, expected: %d\n", len(msgs), messageCtr)
	}
	if len(msgs[1].EmailReceivers) != 0 {
		t.Error("EmailReceivers length should be 0")
	}
	if msgs[2].MessageContent != "" {
		t.Error("MessageContent should be empty string")
	}
}
