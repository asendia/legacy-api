package api

import (
	"context"
	"github.com/asendia/legacy-api/mail"

	"github.com/jackc/pgx/v5"
)

type APIForScheduler struct {
	Context      context.Context
	Tx           pgx.Tx
	SendTelegram func(context.Context, int64, string) error
	SendEmail    func([]mail.MailItem) []mail.SendEmailsResponse
}

func (a *APIForScheduler) sendEmails(items []mail.MailItem) []mail.SendEmailsResponse {
	if a.SendEmail != nil {
		return a.SendEmail(items)
	}
	return mail.SendEmails(items)
}
