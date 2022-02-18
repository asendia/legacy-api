package mail

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
	SendEmail(mails []MailItem) (res []SendEmailsResponse, criticalError error)
}
