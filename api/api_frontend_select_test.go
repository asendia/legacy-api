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
		t.Fatalf("Inconsistent length: %d, expected: %d (ctr), %d (rows)\n", len(msgs), messageCtr, len(rows))
	}
	zeroReceiversCtr := 0
	emptyBodyCtr := 0
	// Order is not guaranteed because the messages are inserted in a transaction,
	// they have the same created_at value
	for i := 0; i < 3; i++ {
		if len(msgs[i].EmailReceivers) == 0 {
			zeroReceiversCtr++
		}
		if msgs[i].MessageContent == "" {
			emptyBodyCtr++
		}
	}
	if zeroReceiversCtr != 1 {
		t.Errorf("1 message should have 0 receivers, but found %d\n", zeroReceiversCtr)
	}
	if emptyBodyCtr != 1 {
		t.Errorf("1 message should have empty body, but found %d\n", emptyBodyCtr)
	}
}
