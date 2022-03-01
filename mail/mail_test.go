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

func TestParseAddress(t *testing.T) {
	email := "isvalid@email.com"
	a, err := ParseAddress(email)
	if err != nil {
		t.Fatalf("The email %s is valid", email)
	}
	if a.Address != email {
		t.Fatalf("Email %s should be equal with %s", email, a.Address)
	}
	email = "doesnthave@dot"
	_, err = ParseAddress(email)
	if err == nil {
		t.Fatalf("The domain in email %s should be invalid", email)
	}
	email = "doesnthavedomain@"
	_, err = ParseAddress(email)
	if err == nil {
		t.Fatalf("The domain in email %s should be invalid", email)
	}
	email = "doesnthaveat"
	_, err = ParseAddress(email)
	if err == nil {
		t.Fatalf("The domain in email %s should be invalid", email)
	}
	email = "@doesnthaveaddress.com"
	_, err = ParseAddress(email)
	if err == nil {
		t.Fatalf("The domain in email %s should be invalid", email)
	}
}

func TestParseAddressList(t *testing.T) {
	addrList, err := ParseAddressList("doesnthave@dot, doesnthaveat, proper@email.com, @doesnhaveaddress")
	if err == nil {
		t.Fatalf("Should have error %+v %+v", addrList, err)
	}
	addrList, err = ParseAddressList("doesnthave@dot, proper@email.com")
	if err == nil {
		t.Fatalf("Should have error %+v %+v", addrList, err)
	}
	addrList, err = ParseAddressList("has@dot.com, proper@email.com,donthavespace@but.proper")
	if err != nil {
		t.Fatalf("Should pass, but %+v %+v", addrList, err)
	}
}
