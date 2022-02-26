package api

import (
	"fmt"
	"net/http"
	"net/mail"
	"os"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/secure"
	"github.com/jackc/pgx/v4"
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
	err = queries.InsertEmailIgnoreConflict(a.Context, jwtRes.Email)
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		return res, err
	}
	row, err := queries.InsertMessageIfLessThanThree(a.Context, data.InsertMessageIfLessThanThreeParams{
		EmailCreator:         jwtRes.Email,
		ContentEncrypted:     encrypted,
		InactivePeriodDays:   param.InactivePeriodDays,
		ReminderIntervalDays: param.ReminderIntervalDays,
		ExtensionSecret:      extensionSecret,
		EmailCreator_2:       jwtRes.Email,
	})
	if err != nil {
		res.StatusCode = http.StatusBadRequest
		return res, err
	}
	for _, emailReceiver := range param.EmailReceivers {
		_, err := mail.ParseAddress(emailReceiver)
		if err != nil {
			res.StatusCode = http.StatusBadRequest
			res.ResponseMsg = "Inavlid receiver email: " + emailReceiver
			return res, err
		}
		addr, err := queries.SelectEmail(a.Context, emailReceiver)
		if err == pgx.ErrNoRows {
		} else if err != nil {
			res.StatusCode = http.StatusInternalServerError
			return res, err
		} else if !addr.IsActive {
			res.StatusCode = http.StatusBadRequest
			res.ResponseMsg = fmt.Sprintf("Email: %s is inactive, please use other emails", emailReceiver)
			return res, err
		}
		err = queries.InsertEmailIgnoreConflict(a.Context, emailReceiver)
		if err != nil {
			res.StatusCode = http.StatusInternalServerError
			return res, err
		}
		extensionSecret, err := secure.GenerateRandomString(ExtensionSecretLength)
		if err != nil {
			res.StatusCode = http.StatusInternalServerError
			return res, err
		}
		_, err = queries.InsertMessagesEmailReceiver(a.Context, data.InsertMessagesEmailReceiverParams{
			MessageID:         row.ID,
			EmailReceiver:     emailReceiver,
			UnsubscribeSecret: extensionSecret,
		})
		if err != nil {
			res.StatusCode = http.StatusInternalServerError
			return res, err
		}
	}
	res.StatusCode = http.StatusOK
	res.ResponseMsg = "Insert successful"
	res.Data = MessageData{
		ID:                   row.ID,
		CreatedAt:            row.CreatedAt,
		EmailCreator:         row.EmailCreator,
		EmailReceivers:       param.EmailReceivers,
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
