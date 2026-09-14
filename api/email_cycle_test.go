package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/asendia/legacy-api/mail"
)

func TestEmailCycleCompletesAfterLastPendingUnsubscribe(t *testing.T) {
	a, msg := telegramTest(t)
	if _, err := a.Tx.Exec(a.Context, `UPDATE messages SET inactive_at=CURRENT_DATE-1 WHERE id=$1`, msg.ID); err != nil {
		t.Fatal(err)
	}
	failed := msg.EmailReceivers[0]
	sends := map[string]int{}
	a.SendEmail = func(items []mail.MailItem) []mail.SendEmailsResponse {
		results := make([]mail.SendEmailsResponse, len(items))
		for i, item := range items {
			email := item.To[0].Email
			sends[email]++
			if email == failed {
				results[i].Err = errors.New("recipient failure")
			}
		}
		return results
	}
	if _, err := a.SendTestamentsOfInactiveMessages(); err != nil {
		t.Fatal(err)
	}
	assertCycle := func(want int, overdue, active bool) {
		t.Helper()
		var count int
		var due, enabled bool
		if err := a.Tx.QueryRow(a.Context, `SELECT sent_counter,inactive_at<CURRENT_DATE,is_active FROM messages WHERE id=$1`, msg.ID).Scan(&count, &due, &enabled); err != nil {
			t.Fatal(err)
		}
		if count != want || due != overdue || enabled != active {
			t.Fatalf("cycle count=%d overdue=%v active=%v; want %d %v %v", count, due, enabled, want, overdue, active)
		}
	}
	assertCycle(0, true, true)
	// A delayed retry must still keep the cycle open.
	if _, err := a.SendTestamentsOfInactiveMessages(); err != nil {
		t.Fatal(err)
	}
	assertCycle(0, true, true)
	var secret string
	if err := a.Tx.QueryRow(a.Context, `SELECT unsubscribe_secret FROM messages_email_receivers WHERE message_id=$1 AND email_receiver=$2`, msg.ID, failed).Scan(&secret); err != nil {
		t.Fatal(err)
	}
	frontend := APIForFrontend{Context: a.Context, Tx: a.Tx}
	res, err := frontend.UnsubscribeMessage(secret, msg.ID)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("unsubscribe: status=%d err=%v", res.StatusCode, err)
	}
	for run := 0; run < 3; run++ {
		if _, err := a.SendTestamentsOfInactiveMessages(); err != nil {
			t.Fatal(err)
		}
	}
	assertCycle(1, false, true)
	for _, email := range msg.EmailReceivers {
		if sends[email] != 1 {
			t.Fatalf("same-cycle duplicate to %s", email)
		}
	}
	// Make later cycles due with distinct dates. Only the subscribed recipient is sent again.
	for cycle := 2; cycle <= 3; cycle++ {
		if _, err := a.Tx.Exec(a.Context, `UPDATE messages SET inactive_at=CURRENT_DATE-$2::integer WHERE id=$1`, msg.ID, cycle); err != nil {
			t.Fatal(err)
		}
		if _, err := a.SendTestamentsOfInactiveMessages(); err != nil {
			t.Fatal(err)
		}
		assertCycle(cycle, false, cycle < 3)
	}
	if sends[failed] != 1 || sends[msg.EmailReceivers[1]] != 3 {
		t.Fatalf("wrong later deliveries: %v", sends)
	}
	if _, err := a.SendTestamentsOfInactiveMessages(); err != nil {
		t.Fatal(err)
	}
	assertCycle(3, false, false)
}

func TestEmailCompletionRequiresCurrentStartedCycle(t *testing.T) {
	for _, test := range []struct {
		name    string
		change  string
		receipt bool
		want    bool
	}{
		{name: "not started", receipt: false},
		{name: "current cycle", receipt: true, want: true},
		{name: "old cycle date", receipt: true, change: `UPDATE email_delivery_receipts SET cycle_date=cycle_date-1`},
		{name: "old extension secret", receipt: true, change: `UPDATE messages SET extension_secret=repeat('x',69)`},
		{name: "inactive message", receipt: true, change: `UPDATE messages SET is_active=false`},
		{name: "future cycle", receipt: true, change: `UPDATE messages SET inactive_at=CURRENT_DATE+1`},
		{name: "empty message", receipt: true, change: `UPDATE messages SET content_encrypted=''`},
		{name: "delivery limit reached", receipt: true, change: `UPDATE messages SET sent_counter=3`},
	} {
		t.Run(test.name, func(t *testing.T) {
			a, msg := telegramTest(t)
			if _, err := a.Tx.Exec(a.Context, `UPDATE messages SET inactive_at=CURRENT_DATE-1 WHERE id=$1`, msg.ID); err != nil {
				t.Fatal(err)
			}
			if _, err := a.Tx.Exec(a.Context, `UPDATE messages_email_receivers SET is_unsubscribed=true WHERE message_id=$1`, msg.ID); err != nil {
				t.Fatal(err)
			}
			if test.receipt {
				if _, err := a.Tx.Exec(a.Context, `INSERT INTO email_delivery_receipts(message_id,email_receiver,extension_secret,cycle_date) SELECT id,$2,extension_secret,inactive_at FROM messages WHERE id=$1`, msg.ID, msg.EmailReceivers[0]); err != nil {
					t.Fatal(err)
				}
			}
			if test.change != "" {
				if _, err := a.Tx.Exec(a.Context, test.change); err != nil {
					t.Fatal(err)
				}
			}
			a.SendEmail = func(items []mail.MailItem) []mail.SendEmailsResponse {
				if len(items) != 0 {
					t.Fatal("unsubscribed recipient was sent")
				}
				return nil
			}
			if _, err := a.SendTestamentsOfInactiveMessages(); err != nil {
				t.Fatal(err)
			}
			var count int
			if err := a.Tx.QueryRow(a.Context, `SELECT sent_counter FROM messages WHERE id=$1`, msg.ID).Scan(&count); err != nil {
				t.Fatal(err)
			}
			want := 0
			if test.want {
				want = 1
			}
			if test.name == "delivery limit reached" {
				want = 3
			}
			if count != want {
				t.Fatalf("got %d completed cycles, want %d", count, want)
			}
		})
	}
}

func TestCompletedEmailCyclesDrainWithoutSendRows(t *testing.T) {
	a, msg := telegramTest(t)
	if _, err := a.Tx.Exec(a.Context, `INSERT INTO messages(email_creator,content_encrypted,extension_secret,inactive_at,next_reminder_at)
  SELECT email_creator,content_encrypted,extension_secret,CURRENT_DATE-1,next_reminder_at
  FROM messages CROSS JOIN generate_series(1,101) WHERE id=$1`, msg.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Tx.Exec(a.Context, `INSERT INTO messages_email_receivers(message_id,email_receiver,is_unsubscribed,unsubscribe_secret)
  SELECT id,$2::text,true,repeat('u',69) FROM messages WHERE email_creator=$3 AND id<>$1`, msg.ID, msg.EmailReceivers[0], msg.EmailCreator); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Tx.Exec(a.Context, `INSERT INTO email_delivery_receipts(message_id,email_receiver,extension_secret,cycle_date)
  SELECT id,$2::text,extension_secret,inactive_at FROM messages WHERE email_creator=$3 AND id<>$1`, msg.ID, msg.EmailReceivers[0], msg.EmailCreator); err != nil {
		t.Fatal(err)
	}
	a.SendEmail = func(items []mail.MailItem) []mail.SendEmailsResponse {
		if len(items) != 0 {
			t.Fatal("completed work was sent again")
		}
		return nil
	}
	for _, want := range []int{100, 101, 101} {
		if _, err := a.SendTestamentsOfInactiveMessages(); err != nil {
			t.Fatal(err)
		}
		var count int
		if err := a.Tx.QueryRow(a.Context, `SELECT count(*) FROM messages WHERE email_creator=$1 AND sent_counter=1`, msg.EmailCreator).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("got %d completed messages, want %d", count, want)
		}
	}
}
