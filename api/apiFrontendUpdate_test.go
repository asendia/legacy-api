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
		res, err := a.InsertMessageV2(generateJwtMessageTemplate(msg.EmailCreator),
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
		res, err := a.InsertMessageV2(generateJwtMessageTemplate(msg.EmailCreator),
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

func TestDiffOldWithNewEmailList(t *testing.T) {
	oldList := []data.MessagesEmailReceiver{
		{EmailReceiver: "a@b"},
	}
	newList := []string{"a@b"}
	actionMap := diffOldWithNewEmailList(oldList, newList)
	if actionMap["a@b"] != "ignore" {
		t.Fatalf("Email should be ignored, but: %s", actionMap["a@b"])
	}

	oldList = receiverList([]string{"a@b", "b@c"})
	newList = []string{"a@b"}
	actionMap = diffOldWithNewEmailList(oldList, newList)
	if actionMap["b@c"] != "delete" {
		t.Fatalf("Email should be deleted, but: %s", actionMap["b@c"])
	}

	oldList = receiverList([]string{"a@b", "b@c"})
	newList = []string{"a@b", "b@c", "c@d"}
	actionMap = diffOldWithNewEmailList(oldList, newList)
	if actionMap["c@d"] != "insert" {
		t.Fatalf("Email should be inserted, but: %s", actionMap["c@d"])
	}

	oldList = receiverList([]string{"a@b", "b@c", "c@d"})
	oldList[0].IsUnsubscribed = true
	newList = []string{"b@c", "c@d", "e@f"}
	actionMap = diffOldWithNewEmailList(oldList, newList)
	if actionMap["a@b"] != "hide" {
		t.Fatalf("Email should be ignored, but: %s", actionMap["a@b"])
	}
	if actionMap["e@f"] != "insert" {
		t.Fatalf("Email should be inserted, but: %s", actionMap["e@f"])
	}

	oldList = receiverList([]string{"a@b", "b@c", "c@d", "d@e"})
	oldList[1].IsUnsubscribed = true
	newList = []string{"a@b", "c@d"}
	actionMap = diffOldWithNewEmailList(oldList, newList)
	if actionMap["a@b"] != "ignore" {
		t.Fatalf("Email should be ignored, but: %s", actionMap["a@b"])
	}
	if actionMap["d@e"] != "delete" {
		t.Fatalf("Email should be deleted, but: %s", actionMap["d@e"])
	}

	oldList = receiverList([]string{"a@b", "b@c", "c@d", "d@e"})
	oldList[1].IsUnsubscribed = true
	newList = []string{}
	actionMap = diffOldWithNewEmailList(oldList, newList)
	if actionMap["a@b"] != "delete" {
		t.Fatalf("Email should be ignored, but: %s", actionMap["a@b"])
	}
	if actionMap["b@c"] != "hide" {
		t.Fatalf("Email should be ignored, but: %s", actionMap["a@b"])
	}
	if actionMap["d@e"] != "delete" {
		t.Fatalf("Email should be deleted, but: %s", actionMap["d@e"])
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
		res, err := a.InsertMessageV2(generateJwtMessageTemplate(msg.EmailCreator),
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
	b.Logf("msg:{update:%d},mail:{insert:%d},rcv:{select:%d,insert:%d,delete:%d}",
		msg_update, mail_insert, rcv_select, rcv_insert, rcv_delete)
}

func BenchmarkUpdateMessageV2(b *testing.B) {
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
		res, err := a.InsertMessageV2(generateJwtMessageTemplate(msg.EmailCreator),
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
		res, err := a.UpdateMessageV2(generateJwtMessageTemplate(row.EmailCreator),
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
	b.Logf("msg:{update:%d},mail:{insert:%d},rcv:{select:%d,insert:%d,delete:%d}",
		msg_update, mail_insert, rcv_select, rcv_insert, rcv_delete)
}
