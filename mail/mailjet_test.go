package mail

import (
	"errors"
	"os"
	"testing"

	"github.com/asendia/legacy-api/simple"
	mailjet "github.com/mailjet/mailjet-apiv3-go"
)

func TestSendEmails(t *testing.T) {
	param := ReminderEmailParams{
		Title:          "Reminder to extend the delivery schedule of warisin.com testament",
		FullName:       "Warisin Team",
		InactiveAt:     simple.TimeTodayUTC().Add(simple.DaysToDuration(90)).Local().Format("2006-01-02"),
		EmailReceivers: []string{"asendia@warisin.com", "noreply@warisin.com"},
		ExtensionURL:   "https://warisin.com/extend?id=some-id&secret=some-secret"}
	htmlContent, err := GenerateReminderEmail(param)
	if err != nil {
		t.Fatalf("Cannot generate email from template: %v", err)
	}
	mails := []mailjet.InfoMessagesV31{
		{
			From: &mailjet.RecipientV31{
				Email: "noreply@warisin.com",
				Name:  "Warisin Team",
			},
			To: &mailjet.RecipientsV31{
				mailjet.RecipientV31{
					Email: "asendia@warisin.com",
					Name:  "Warisin User",
				},
			},
			Subject:  param.Title,
			HTMLPart: htmlContent,
			CustomID: "legacy-reminder",
		},
	}
	publicKey := os.Getenv("MAILJET_PUBLIC_KEY")
	privateKey := os.Getenv("MAILJET_PRIVATE_KEY")
	res, err := SendEmails(publicKey, privateKey, mails)
	if errors.Is(err, ErrMailjetNoAPIKeys) {
		t.Log(err)
		return
	} else if err != nil {
		t.Fatalf("Cannot send emails: %+v\n", err)
	}
	t.Logf("Sending emails successful: %+v\n", res)
}
