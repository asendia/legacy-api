package mail

import (
	"errors"
	"log"

	"github.com/mailjet/mailjet-apiv3-go/v4"
)

type Mailjet struct {
	APIKey      string
	SecretKey   string
	SandboxMode bool
}

func (m *Mailjet) GetVendorID() string {
	return "MAILJET"
}

func (m *Mailjet) HasAPIKey() bool {
	return m.APIKey != "" && m.SecretKey != ""
}

func (m *Mailjet) SendEmails(mails []MailItem) (res []SendEmailsResponse, criticalError error) {
	if !m.HasAPIKey() {
		return res, ErrMailNoAPIKey
	}
	client := mailjet.NewMailjetClient(m.APIKey, m.SecretKey)
	messages := mailjet.MessagesV31{Info: convertMailItemsToMailjet(mails),
		SandBoxMode: m.SandboxMode}
	_, criticalError = client.SendMailV31(&messages)
	errFeedbackList := &mailjet.APIFeedbackErrorsV31{}
	isErrFeedbacklist := errors.As(criticalError, &errFeedbackList)
	if isErrFeedbacklist {
		// Do nothing
	} else if criticalError != nil {
		log.Printf("Critical, cannot send emails: %+v\n", criticalError)
		return res, criticalError
	}
	for id, mail := range mails {
		emailRes := SendEmailsResponse{VendorID: m.GetVendorID()}
		if mail.To != nil {
			for _, m := range mail.To {
				emailRes.Emails = append(emailRes.Emails, m.Email)
			}
		}
		isError := isErrFeedbacklist &&
			len(errFeedbackList.Messages) > id &&
			len(errFeedbackList.Messages[id].Errors) > 0
		if isError {
			// TODO: how to handle this error properly
			emailRes.Err = errors.New(errFeedbackList.Messages[id].Errors[0].ErrorMessage)
		}
		res = append(res, emailRes)
	}
	return res, nil
}

func convertMailItemsToMailjet(mails []MailItem) (mailjetMails []mailjet.InfoMessagesV31) {
	for _, m := range mails {
		mailjetTo := mailjet.RecipientsV31{}
		for _, t := range m.To {
			mailjetTo = append(mailjetTo, mailjet.RecipientV31{
				Email: t.Email,
				Name:  t.Name,
			})
		}
		mailjetMails = append(mailjetMails, mailjet.InfoMessagesV31{
			From: &mailjet.RecipientV31{
				Email: m.From.Email,
				Name:  m.From.Name,
			},
			To:       &mailjetTo,
			Subject:  m.Subject,
			HTMLPart: m.HtmlContent,
			CustomID: "legacy-reminder",
		})
	}
	return
}
