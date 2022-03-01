package mail

import (
	"errors"
	"os"
	"testing"
)

func TestSendgridMultipleEmailsMultipleTos(t *testing.T) {
	s := Sendgrid{APIKey: os.Getenv("SENDGRID_API_KEY"),
		SandboxMode: os.Getenv("ENVIRONMENT") != "prod"}
	toList := []string{"asendia@warisin.com", "should@beinvalid", "test@warisin.com"}
	mails, err := generateMultipleEmailsMultipleTos(toList, s.GetVendorID())
	if err != nil {
		t.Fatalf("%+v", err)
	}
	res, err := s.SendEmails(mails)
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

func TestSendgridSingleEmailMultipleTos(t *testing.T) {
	s := Sendgrid{APIKey: os.Getenv("SENDGRID_API_KEY"),
		SandboxMode: os.Getenv("ENVIRONMENT") != "prod"}
	toList := []string{"asendia@warisin.com", "invalid@emailformat", "test@warisin.com"}
	mails, err := generateSingleEmailMultipleTos(toList, s.GetVendorID())
	if err != nil {
		t.Fatalf("%+v", err)
	}
	res, err := s.SendEmails(mails)
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
