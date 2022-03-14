package mail

import (
	"errors"
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
	toList := []string{"test@sejiwo.com", "invalid@format", "asendia@sejiwo.com"}
	mails := []MailItem{}
	for id, to := range toList {
		param := ReminderEmailParams{
			Title:      "Reminder to extend the delivery schedule of sejiwo.com testament",
			FullName:   "Warisin Team",
			InactiveAt: simple.TimeTodayUTC().Add(simple.DaysToDuration(90)).Local().Format("2006-01-02"),
			TestamentReceivers: []string{
				fmt.Sprintf("testamentreceiver-%d-1@somedomain.com", id),
				fmt.Sprintf("testamentreceiver-%d-2@somedomain.com", id),
			},
			ExtensionURL: "https://sejiwo.com/extend?id=some-id&secret=some-secret"}
		htmlContent, err := GenerateReminderEmail(param)
		if err != nil {
			t.Fatalf("Cannot generate email from template: %v", err)
		}
		mails = append(mails, MailItem{
			From: MailAddress{
				Email: "noreply@sejiwo.com",
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

func generateMultipleEmailsMultipleTos(toList []string, vendorID string) ([]MailItem, error) {
	mails := []MailItem{}
	for id, to := range toList {
		param := ReminderEmailParams{
			Title:      "Reminder to extend the delivery schedule of sejiwo.com testament from " + vendorID,
			FullName:   "Warisin Team",
			InactiveAt: simple.TimeTodayUTC().Add(simple.DaysToDuration(90)).Local().Format("2006-01-02"),
			TestamentReceivers: []string{
				fmt.Sprintf("testamentreceiver-%d-1@somedomain.com", id),
				fmt.Sprintf("testamentreceiver-%d-2@somedomain.com", id),
			},
			ExtensionURL: "https://sejiwo.com/extend?id=some-id&secret=some-secret"}
		htmlContent, err := GenerateReminderEmail(param)
		if err != nil {
			return mails, errors.New(fmt.Sprintf("Cannot generate email from template: %v", err))
		}
		mails = append(mails, MailItem{
			From: MailAddress{
				Email: "noreply@sejiwo.com",
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
	return mails, nil
}

func generateSingleEmailMultipleTos(toList []string, vendorID string) ([]MailItem, error) {
	mails := []MailItem{}
	param := TestamentEmailParams{
		Title:                 "Reminder to extend the delivery schedule of sejiwo.com testament from " + vendorID,
		FullName:              "Warisin Team",
		EmailCreator:          "noreply@sejiwo.com",
		MessageContentPerLine: []string{"Line 1", "Line 2", "Line 3"},
		UnsubscribeURL:        "https://sejiwo.com/?action=unsubscribe&id=some-id&secret=some-secret"}
	htmlContent, err := GenerateTestamentEmail(param)
	if err != nil {
		return mails, errors.New(fmt.Sprintf("Cannot generate email from template: %v", err))
	}
	tos := []MailAddress{}
	for id, mt := range toList {
		tos = append(tos, MailAddress{Email: mt, Name: fmt.Sprintf("User %d", id)})
	}
	mails = append(mails, MailItem{
		From: MailAddress{
			Email: "noreply@sejiwo.com",
			Name:  "Warisin Team",
		},
		To:          tos,
		Subject:     param.Title,
		HtmlContent: htmlContent,
	})
	return mails, nil
}
