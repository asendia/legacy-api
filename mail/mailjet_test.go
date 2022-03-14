package mail

import (
	"errors"
	"os"
	"testing"

	"github.com/asendia/legacy-api/simple"
)

func TestMailjetSingleEmailSingleTo(t *testing.T) {
	param := ReminderEmailParams{
		Title:              "Reminder to extend the delivery schedule of sejiwo.com testament",
		FullName:           "Sejiwo Team",
		InactiveAt:         simple.TimeTodayUTC().Add(simple.DaysToDuration(90)).Local().Format("2006-01-02"),
		TestamentReceivers: []string{"test@sejiwo.com", "noreply@sejiwo.com"},
		ExtensionURL:       "https://sejiwo.com/extend?id=some-id&secret=some-secret"}
	htmlContent, err := GenerateReminderEmail(param)
	if err != nil {
		t.Fatalf("Cannot generate email from template: %v", err)
	}
	mails := []MailItem{
		{
			From: MailAddress{
				Email: "noreply@sejiwo.com",
				Name:  "Sejiwo Team",
			},
			To: []MailAddress{
				{
					Email: "test@sejiwo.com",
					Name:  "Sejiwo User",
				},
			},
			Subject:     param.Title,
			HtmlContent: htmlContent,
		},
	}
	m := Mailjet{APIKey: os.Getenv("MAILJET_API_KEY"),
		SecretKey:   os.Getenv("MAILJET_SECRET_KEY"),
		SandboxMode: os.Getenv("ENVIRONMENT") != "prod"}
	res, err := m.SendEmails(mails)
	if errors.Is(err, ErrMailNoAPIKey) {
		t.Logf("%+v", err)
		return
	} else if err != nil {
		t.Fatalf("Cannot send emails: %+v\n", err)
	}
	t.Logf("Sending emails successful: %+v\n", res)
}

func TestMailjetMultipleEmailsMultipleTos(t *testing.T) {
	m := Mailjet{APIKey: os.Getenv("MAILJET_API_KEY"),
		SecretKey:   os.Getenv("MAILJET_SECRET_KEY"),
		SandboxMode: os.Getenv("ENVIRONMENT") != "prod"}
	toList := []string{"asendia@sejiwo.com", "should@beinvalid", "test@sejiwo.com"}
	mails, err := generateMultipleEmailsMultipleTos(toList, m.GetVendorID())
	if err != nil {
		t.Fatalf("%+v", err)
	}
	res, err := m.SendEmails(mails)
	if errors.Is(err, ErrMailNoAPIKey) {
		t.Logf("%+v", err)
		return
	}
	if err != nil {
		t.Fatalf("Sendgrid error %+v\n", err)
	}
	if res[1].Err == nil {
		t.Fatal("Email should be invalid")
	}
}

func TestMailjetSingleEmailMultipleTos(t *testing.T) {
	m := Mailjet{APIKey: os.Getenv("MAILJET_API_KEY"),
		SecretKey:   os.Getenv("MAILJET_SECRET_KEY"),
		SandboxMode: os.Getenv("ENVIRONMENT") != "prod"}
	toList := []string{"asendia@sejiwo.com", "invalid@emailformat", "test@sejiwo.com"}
	mails, err := generateSingleEmailMultipleTos(toList, m.GetVendorID())
	if err != nil {
		t.Fatalf("%+v", err)
	}
	res, err := m.SendEmails(mails)
	if errors.Is(err, ErrMailNoAPIKey) {
		t.Logf("%+v", err)
		return
	}
	if err != nil {
		t.Fatalf("Sendgrid error %+v\n", err)
	}
	if res[0].Err == nil {
		t.Fatal("Email should be invalid")
	}
}
