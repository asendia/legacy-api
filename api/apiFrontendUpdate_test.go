package api

import (
	"testing"

	"github.com/asendia/legacy-api/data"
)

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
