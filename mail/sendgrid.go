package mail

import (
	"errors"
	"log"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type Sendgrid struct {
	PrivateKey string
}

func (s *Sendgrid) SendEmails(mails []MailItem) (res []SendEmailsResponse, criticalError error) {
	if s.PrivateKey == "" {
		return res, errors.New("Sendgrid no Private Key")
	}
	sendgridMail := mail.NewV3Mail()
	content := mail.NewContent("text/html", "%htmlContent%")
	sendgridMail.AddContent(content)
	for id, m := range mails {
		if id == 0 {
			sendgridMail.SetFrom(mail.NewEmail(m.From.Name, m.From.Email))
		}
		p := mail.NewPersonalization()
		for _, t := range m.To {
			p.AddTos(mail.NewEmail(t.Name, t.Email))
		}
		p.SetSubstitution("%htmlContent%", m.HtmlContent)
		p.From = mail.NewEmail(m.From.Name, m.From.Email)
		p.Subject = m.Subject
		sendgridMail.AddPersonalizations(p)
	}
	request := sendgrid.GetRequest(s.PrivateKey, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	request.Body = mail.GetRequestBody(sendgridMail)
	response, err := sendgrid.API(request)
	if err != nil {
		log.Printf("Sendgrid error: %+v", err)
		return res, err
	}
	log.Println(response.StatusCode)
	log.Println(response.Body)
	log.Println(response.Headers)
	for _, m := range mails {
		for _, t := range m.To {
			res = append(res, SendEmailsResponse{Err: nil, Email: t.Email})
		}
	}
	return res, err
}
