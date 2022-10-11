package api

import (
	"net/http"
	"net/mail"
	"os"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/secure"
)

func (a *APIForFrontend) InsertMessage(jwtRes secure.JWTResponse, param APIParamInsertMessage) (res APIResponse, err error) {
	extensionSecret, err := secure.GenerateRandomString(ExtensionSecretLength)
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		return res, err
	}
	encrypted, err := EncryptMessageContent(param.MessageContent, os.Getenv("ENCRYPTION_KEY"))
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		return res, err
	}
	queries := data.New(a.Tx)
	row, err := queries.InsertMessage(a.Context, data.InsertMessageParams{
		EmailCreator:         jwtRes.Email,
		ContentEncrypted:     encrypted,
		InactivePeriodDays:   param.InactivePeriodDays,
		ReminderIntervalDays: param.ReminderIntervalDays,
		ExtensionSecret:      extensionSecret,
	})
	if err != nil {
		res.StatusCode = http.StatusBadRequest
		return res, err
	}
	emailReceivers := []string{}
	unsubscribeSecrets := []string{}
	for _, emailReceiver := range param.EmailReceivers {
		_, err := mail.ParseAddress(emailReceiver)
		if err != nil {
			res.StatusCode = http.StatusBadRequest
			res.ResponseMsg = "Inavlid receiver email: " + emailReceiver
			return res, err
		}
		unsubscribeSecret, err := secure.GenerateRandomString(ExtensionSecretLength)
		if err != nil {
			res.StatusCode = http.StatusInternalServerError
			return res, err
		}
		emailReceivers = append(emailReceivers, emailReceiver)
		unsubscribeSecrets = append(unsubscribeSecrets, unsubscribeSecret)
	}
	rcvRows, err := queries.UpsertReceivers(a.Context, data.UpsertReceiversParams{
		MessageID:          row.ID,
		EmailReceivers:     emailReceivers,
		UnsubscribeSecrets: unsubscribeSecrets,
	})
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		return res, err
	}
	emailReceivers = []string{}
	for _, emailReceiver := range rcvRows {
		emailReceivers = append(emailReceivers, emailReceiver.EmailReceiver)
	}
	res.StatusCode = http.StatusOK
	res.ResponseMsg = "Insert successful"
	res.Data = MessageData{
		ID:                   row.ID,
		CreatedAt:            row.CreatedAt,
		EmailCreator:         row.EmailCreator,
		EmailReceivers:       emailReceivers,
		MessageContent:       param.MessageContent,
		InactivePeriodDays:   row.InactivePeriodDays,
		ReminderIntervalDays: row.ReminderIntervalDays,
		IsActive:             row.IsActive,
		ExtensionSecret:      row.ExtensionSecret,
		InactiveAt:           row.InactiveAt,
		NextReminderAt:       row.NextReminderAt,
	}
	return res, err
}
