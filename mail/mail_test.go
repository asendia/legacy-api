package mail

import (
	"fmt"
	"os"
	"testing"

	"github.com/asendia/legacy-api/simple"
)

func TestMain(m *testing.M) {
	simple.MustLoadEnv("../.env-test.yaml")
	code := m.Run()
	os.Exit(code)
}

func TestSendEmailsMultipleEmailsMultipleTos(t *testing.T) {
	toList := []string{"test@warisin.com", "invalid@format", "asendia@warisin.com"}
	mails := []MailItem{}
	for id, to := range toList {
		param := ReminderEmailParams{
			Title:      "Reminder to extend the delivery schedule of warisin.com testament",
			FullName:   "Warisin Team",
			InactiveAt: simple.TimeTodayUTC().Add(simple.DaysToDuration(90)).Local().Format("2006-01-02"),
			TestamentReceivers: []string{
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
					Email: to,
					Name:  fmt.Sprintf("Warisin User %d", id),
				},
			},
			Subject:     param.Title,
			HtmlContent: htmlContent,
		})
	}
	res := SendEmails(mails)
	for id, r := range res {
		if r.Err != nil {
			fmt.Printf("SendEmails %d failed: %+v", id, r.Err)
		}
	}
}
