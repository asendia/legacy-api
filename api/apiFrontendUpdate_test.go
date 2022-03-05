package api

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/asendia/legacy-api/data"
)

func TestUpdateMessage(t *testing.T) {
	ctx := context.Background()
	tx, err := pgxPoolConn.Begin(ctx)
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
				EmailReceivers: []string{
					"email-" + strconv.Itoa(id) + "-1@warisin.com",
					"email-" + strconv.Itoa(id) + "-2@warisin.com",
				},
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
		} else if msg.EmailReceivers[1] != "email-"+strconv.Itoa(id)+"-2@warisin.com" {
			t.Error("UpdateMessage failed to change EmailReceivers value\n")
		} else if row.EmailReceivers[0] == msg.EmailReceivers[0] {
			t.Error("UpdateMessage failed to change EmailReceivers value\n")
		}
	}
}

func TestUpdateMessageNoReceiver(t *testing.T) {
	ctx := context.Background()
	tx, err := pgxPoolConn.Begin(ctx)
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
				EmailReceivers:       []string{},
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
		} else if len(msg.EmailReceivers) != 0 {
			t.Error("EmailReceivers length should be 0")
		}
	}
}

func receiverList(emailList []string) []data.MessagesEmailReceiver {
	msgList := []data.MessagesEmailReceiver{}
	for _, email := range emailList {
		msgList = append(msgList, data.MessagesEmailReceiver{
			EmailReceiver: email,
		})
	}
	return msgList
}

func BenchmarkUpdateMessage(b *testing.B) {
	ctx := context.Background()
	tx, err := pgxPoolConn.Begin(ctx)
	if err != nil {
		b.Errorf("Cannot begin transaction: %v\n", err)
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
			b.Errorf("Insert failed: %v\n", err)
			return
		}
		rows = append(rows, res.Data.(MessageData))
	}
	for i := 0; i < b.N; i++ {
		id := i % len(rows)
		row := rows[id]
		additionalMessage := ""
		if i%3 == 0 {
			additionalMessage = " ADDITIONAL TEXT"
		}
		row.IsActive = id%2 == 0
		for j := 0; j < len(rows[id].EmailReceivers); j++ {
			rows[id].EmailReceivers[j] = "email-" + strconv.Itoa(i) + "-" + strconv.Itoa(j) + "@warisin.com"
		}
		res, err := a.UpdateMessage(generateJwtMessageTemplate(row.EmailCreator),
			APIParamUpdateMessage{
				MessageContent:       row.MessageContent + additionalMessage,
				InactivePeriodDays:   row.InactivePeriodDays,
				ReminderIntervalDays: row.ReminderIntervalDays,
				IsActive:             row.IsActive,
				ExtensionSecret:      row.ExtensionSecret,
				ID:                   row.ID,
				EmailReceivers:       rows[id].EmailReceivers,
			})
		if err != nil {
			b.Errorf("Update failed: %v\n", err)
			return
		}
		msg := res.Data.(MessageData)
		if row.IsActive != msg.IsActive {
			b.Error("UpdateMessage failed to change IsActive value\n")
		}
	}
}
