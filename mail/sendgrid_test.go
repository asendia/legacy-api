package mail

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/asendia/legacy-api/simple"
)

func TestSendgridMultipleEmailsMultipleTos(t *testing.T) {
	s := Sendgrid{PrivateKey: os.Getenv("SENDGRID_PRIVATE_KEY")}
	mails := []MailItem{}
	toList := []string{"asendia@warisin.com", "should@beinvalid", "test@warisin.com"}
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

func TestSendgridSingleEmailMultipleTos(t *testing.T) {
	s := Sendgrid{PrivateKey: os.Getenv("SENDGRID_PRIVATE_KEY")}
	mails := []MailItem{}
	toList := []string{"asendia@warisin.com", "invalid@emailformat", "test@warisin.com"}
	param := TestamentEmailParams{
		Title:                 "Reminder to extend the delivery schedule of warisin.com testament",
		FullName:              "Warisin Team",
		EmailCreator:          "noreply@warisin.com",
		MessageContentPerLine: []string{"Line 1", "Line 2", "Line 3"},
		UnsubscribeURL:        "https://warisin.com/?action=unsubscribe&id=some-id&secret=some-secret"}
	htmlContent, err := GenerateTestamentEmail(param)
	if err != nil {
		t.Fatalf("Cannot generate email from template: %v", err)
	}
	tos := []MailAddress{}
	for id, mt := range toList {
		tos = append(tos, MailAddress{Email: mt, Name: fmt.Sprintf("User %d", id)})
	}
	mails = append(mails, MailItem{
		From: MailAddress{
			Email: "noreply@warisin.com",
			Name:  "Warisin Team",
		},
		To:          tos,
		Subject:     param.Title,
		HtmlContent: htmlContent,
	})
	_, err = s.SendEmails(mails)
	if errors.Is(err, ErrSendgridNoPrivateKey) {
		t.Logf("Please specify SENDGRID_PRIVATE_KEY: %+v", err)
		return
	}
	if err != nil {
		t.Fatalf("Sendgrid error %+v\n", err)
	}
}
