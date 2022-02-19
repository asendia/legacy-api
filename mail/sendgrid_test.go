package mail

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/asendia/legacy-api/simple"
)

func TestSendgrid(t *testing.T) {
	s := Sendgrid{PrivateKey: os.Getenv("SENDGRID_PRIVATE_KEY")}
	mails := []MailItem{}
	rcvrs := []string{"swiftyoshioka@gmail.com", "shouldbeinvalid", "test@warisin.com"}
	for id, rcvr := range rcvrs {
		param := ReminderEmailParams{
			Title:      "Reminder to extend the delivery schedule of warisin.com testament",
			FullName:   "Warisin Team",
			InactiveAt: simple.TimeTodayUTC().Add(simple.DaysToDuration(90)).Local().Format("2006-01-02"),
			EmailReceivers: []string{
				fmt.Sprintf("testamentreceiver-%d-1@somedomain.com", id),
				fmt.Sprintf("testamentreceiver-%d-2@somedomain.com", id),
			},
			ExtensionURL: "https://warisin.com/extend?id=some-id&secret=some-secret"}
		htmlContent, err := GenerateReminderEmail(param)
		if err != nil {
			t.Fatalf("Cannot generate email from template: %v", err)
		}
		mails = append(mails, MailItem{
			From: MailAddress{
				Email: "noreply@warisin.com",
				Name:  "Warisin Team",
			},
			To: []MailAddress{
				{
					Email: rcvr,
					Name:  fmt.Sprintf("Warisin User %d", id),
				},
			},
			Subject:     param.Title,
			HtmlContent: htmlContent,
		})
	}
	res, err := s.SendEmails(mails)
	if errors.Is(err, ErrSendgridNoPrivateKey) {
		t.Logf("Please specify SENDGRID_PRIVATE_KEY: %+v", err)
		return
	}
	if err != nil {
		t.Fatalf("Sendgrid error %+v\n", err)
	}
	if res[1].Err == nil {
		t.Fatal("Email should be invalid")
	}
}
