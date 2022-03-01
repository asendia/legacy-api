package mail

import (
	"errors"
	"fmt"
	"math"
	"net/mail"
	"os"
	"strings"
)

type MailAddress struct {
	Name  string
	Email string
}

type MailItem struct {
	From        MailAddress
	To          []MailAddress
	Subject     string
	HtmlContent string
}

type Mail interface {
	SendEmails(mails []MailItem) (res []SendEmailsResponse, criticalError error)
	HasAPIKey() bool
	GetVendorID() string
}

type SendEmailsResponse struct {
	Err      error
	Emails   []string
	VendorID string
}

type SendMailConfig struct {
	Vendors []SendMailVendorConfig
}
type SendMailVendorConfig struct {
	Vendor     Mail
	DailyLimit int
}

// Send emails using multiple vendors
func SendEmails(mails []MailItem) (res []SendEmailsResponse) {
	vendorMailjet := Mailjet{APIKey: os.Getenv("MAILJET_API_KEY"), SecretKey: os.Getenv("MAILJET_SECRET_KEY")}
	vendorSendgrid := Sendgrid{APIKey: os.Getenv("SENDGRID_API_KEY")}
	cfg := SendMailConfig{
		Vendors: []SendMailVendorConfig{
			{
				Vendor:     &vendorMailjet,
				DailyLimit: 200,
			},
			{
				Vendor:     &vendorSendgrid,
				DailyLimit: 100,
			},
		},
	}
	totalDailyLimit := 0
	for _, v := range cfg.Vendors {
		if v.Vendor.HasAPIKey() {
			totalDailyLimit += v.DailyLimit
		}
	}
	currentMailIndex := 0
	// Distribute emails across vendors based on DailyLimit
	for id, v := range cfg.Vendors {
		if !v.Vendor.HasAPIKey() {
			continue
		}
		// Break if there is no more email to send
		if currentMailIndex >= len(mails) {
			break
		}
		totalMailsForThisVendor := int(math.Floor(float64(v.DailyLimit) / float64(totalDailyLimit) * float64(len(mails))))
		targetMailIndex := currentMailIndex + totalMailsForThisVendor
		// If target exceeds the emails length, or it is the last vendor
		if targetMailIndex > len(mails) || id+1 == len(cfg.Vendors) {
			targetMailIndex = len(mails)
		}
		emailsUsingThisVendor := mails[currentMailIndex:targetMailIndex]
		r, cErr := v.Vendor.SendEmails(emailsUsingThisVendor)
		if cErr != nil {
			for _, email := range emailsUsingThisVendor {
				emailTos := []string{}
				for _, mt := range email.To {
					emailTos = append(emailTos, mt.Email)
				}
				res = append(res, SendEmailsResponse{
					Err:    errors.New(fmt.Sprintf("Critical error from vendor id: %d", id)),
					Emails: emailTos,
				})
			}
		}
		res = append(res, r...)
		currentMailIndex = targetMailIndex
	}
	return res
}

func ParseAddress(address string) (addr *mail.Address, err error) {
	addr, err = mail.ParseAddress(address)
	if err != nil {
		return
	}
	err = parseValidAddress(addr.Address)
	return
}

func ParseAddressList(list string) (addrList []*mail.Address, err error) {
	addrList, err = mail.ParseAddressList(list)
	if err != nil {
		return
	}
	for id, addr := range addrList {
		err := parseValidAddress(addr.Address)
		if err != nil {
			return addrList,
				errors.New(fmt.Sprintf("Email: %s at index %d is invalid with error: %+v", addr.Address, id, err))
		}
	}
	return
}

func parseValidAddress(validAddress string) error {
	atPos := strings.LastIndex(validAddress, "@")
	domainName := validAddress[atPos+1:]
	if !strings.Contains(domainName, ".") {
		return errors.New("Email domain doesn't contain dot (.)")
	}
	return nil
}
