package mail

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type Sendgrid struct {
	APIKey string
}

func (m *Sendgrid) GetVendorID() string {
	return "SENDGRID"
}

func (s *Sendgrid) HasAPIKey() bool {
	return s.APIKey != ""
}

func (s *Sendgrid) SendEmails(mails []MailItem) (res []SendEmailsResponse, criticalError error) {
	if !s.HasAPIKey() {
		return res, ErrSendgridNoPrivateKey
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
	client := sendgrid.NewSendClient(s.APIKey)
	sgRes, err := client.Send(sendgridMail)
	if err != nil {
		log.Printf("Sendgrid error: %+v", err)
		return res, err
	}

	// Parse errDesc
	// Example JSON value:
	// {
	//   "field": "personalizations.1.to.0.email",
	//   "help": "http://sendgrid.com/docs/API_Reference/Web_API_v3/Mail/errors.html#message.personalizations.to",
	//   "message": "Does not contain a valid address."
	// }
	sgErrRes := SendgridErrorResponse{}
	if sgRes.StatusCode < 200 || sgRes.StatusCode >= 300 {
		log.Printf("Sendgrid error statusCode: %d, body: %+v", sgRes.StatusCode, sgRes.Body)
		jsonErr := json.Unmarshal([]byte(sgRes.Body), &sgErrRes)
		if jsonErr != nil {
			return res, jsonErr
		}
		// Sendgrid won't deliver any emails if one input is invalid
		err = errors.New("Sendgrid received non 200 status code")
	}
	errMap := map[int]map[int]string{}
	for _, errDesc := range sgErrRes.Errors {
		ss := strings.Split(errDesc.Field, ".")
		if len(ss) != 5 || ss[0] != "personalizations" || ss[2] != "to" || ss[4] != "email" {
			log.Printf("Unknown sendgrid errDesc.Flied: %s", errDesc.Field)
			continue
		}
		pID, err := strconv.Atoi(ss[1])
		if err != nil {
			log.Printf("Failed to parse errDesc.Field personalization ID: %s", ss[1])
			continue
		}
		toID, err := strconv.Atoi(ss[3])
		if err != nil {
			log.Printf("Failed to parse errDesc.Field to email: %s", ss[3])
			continue
		}
		if errMap[pID] == nil {
			errMap[pID] = map[int]string{}
		}
		errMap[pID][toID] = errDesc.Message
	}
	for pID, m := range mails {
		for toID, t := range m.To {
			var err error
			if errMap[pID][toID] != "" {
				err = errors.New(errMap[pID][toID])
			}
			if pID >= len(res) {
				res = append(res, SendEmailsResponse{
					Err: err, Emails: []string{t.Email}, VendorID: s.GetVendorID()})
			} else {
				res[pID].Emails = append(res[pID].Emails, t.Email)
			}
		}
	}
	return res, err
}

type SendgridErrorResponse struct {
	Errors []SendgridErrorDescription `json:"errors"`
}

type SendgridErrorDescription struct {
	Field   string `json:"field"`
	Help    string `json:"help"`
	Message string `json:"message"`
}

var ErrSendgridNoPrivateKey = errors.New("Sendgrid no Private Key")
