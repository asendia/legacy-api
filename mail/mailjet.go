package mail

import (
	"errors"
	"log"

	mailjet "github.com/mailjet/mailjet-apiv3-go"
)

func SendEmails(publicKey string, privateKey string, mails []mailjet.InfoMessagesV31) (res []SendEmailsResponse, criticalError error) {
	if publicKey == "" || privateKey == "" {
		return res, ErrMailjetNoAPIKeys
	}
	m := mailjet.NewMailjetClient(publicKey, privateKey)
	messages := mailjet.MessagesV31{Info: mails}
	_, criticalError = m.SendMailV31(&messages)
	errFeedbackList := &mailjet.APIFeedbackErrorsV31{}
	isErrFeedbacklist := errors.As(criticalError, &errFeedbackList)
	if isErrFeedbacklist {
		// Do nothing
	} else if criticalError != nil {
		log.Printf("Critical, cannot send emails: %+v\n", criticalError)
		return res, criticalError
	}
	for id, mail := range mails {
		emailRes := SendEmailsResponse{}
		if mail.To != nil && len(*mail.To) > 0 {
			emailRes.Email = (*mail.To)[0].Email
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

type SendEmailsResponse struct {
	Err   error
	Email string
}

var ErrMailjetNoAPIKeys = errors.New("Mailjet requires publicKey & privateKey, set in the " +
	"MAILJET_PUBLIC_KEY & MAILJET_PRIVATE_KEY environment variable")
