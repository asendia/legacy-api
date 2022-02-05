package api

import (
	"context"
	"net/http"
	"net/mail"
	"os"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/secure"
	"github.com/asendia/legacy-api/simple"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

type APIForFrontend struct {
	Context context.Context
	Tx      pgx.Tx
}

func (a *APIForFrontend) InsertMessage(msg MessageData) (res APIResponse, err error) {
	if msg.NextReminderAt.IsZero() {
		msg.NextReminderAt = simple.TimeTodayUTC().Add(simple.DaysToDuration(int(msg.ReminderIntervalDays)))
	}
	if msg.InactiveAt.IsZero() {
		msg.InactiveAt = simple.TimeTodayUTC().Add(simple.DaysToDuration(int(msg.InactivePeriodDays)))
	}
	msg.ExtensionSecret, err = secure.GenerateRandomString(ExtensionSecretLength)
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		return res, err
	}
	encrypted, err := EncryptMessageContent(msg.MessageContent, os.Getenv("ENCRYPTION_KEY"))
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		return res, err
	}
	queries := data.New(a.Tx)
	res.Data, err = queries.InsertMessage(a.Context, data.InsertMessageParams{
		EmailCreator:         msg.EmailCreator,
		EmailReceivers:       msg.EmailReceivers,
		MessageContent:       encrypted,
		InactivePeriodDays:   msg.InactivePeriodDays,
		ReminderIntervalDays: msg.ReminderIntervalDays,
		IsActive:             msg.IsActive,
		ExtensionSecret:      msg.ExtensionSecret,
		InactiveAt:           msg.InactiveAt,
		NextReminderAt:       msg.NextReminderAt,
	})
	if err != nil {
		res.StatusCode = http.StatusBadRequest
		return res, err
	}
	res.StatusCode = http.StatusOK
	res.ResponseMsg = "Insert successful"
	return res, err
}

func (a *APIForFrontend) SelectMessageByID(id uuid.UUID) (res APIResponse, err error) {
	queries := data.New(a.Tx)
	row, err := queries.SelectMessageByID(a.Context, id)
	if err != nil {
		res.StatusCode = http.StatusNotFound
		return res, err
	}
	msgContent, err := DecryptMessageContent(row.MessageContent, os.Getenv("ENCRYPTION_KEY"))
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		return
	}
	res.StatusCode = http.StatusOK
	res.ResponseMsg = "Select message successful"
	row.MessageContent = msgContent
	res.Data = row
	return res, err
}

func (a *APIForFrontend) SelectMessagesByEmailCreator(emailCreator string) (res APIResponse, err error) {
	queries := data.New(a.Tx)
	if _, err = mail.ParseAddress(emailCreator); err != nil {
		res.StatusCode = http.StatusBadRequest
		res.ResponseMsg = err.Error()
		return res, err
	}
	rows, err := queries.SelectMessagesByEmailCreator(a.Context, emailCreator)
	if err != nil {
		return res, err
	}
	for _, row := range rows {
		row.MessageContent, err = DecryptMessageContent(row.MessageContent, os.Getenv("ENCRYPTION_KEY"))
		if err != nil {
			return res, err
		}
	}
	res.Data = rows
	res.StatusCode = http.StatusOK
	res.ResponseMsg = "Select messages successful"
	return res, err
}

func (a *APIForFrontend) UpdateMessage(param data.UpdateMessageParams) (res APIResponse, err error) {
	queries := data.New(a.Tx)
	// Refresh extension secret on every update
	param.ExtensionSecret, err = secure.GenerateRandomString(ExtensionSecretLength)
	if err != nil {
		return res, err
	}
	param.MessageContent, err = EncryptMessageContent(param.MessageContent, os.Getenv("ENCRYPTION_KEY"))
	if err != nil {
		return res, err
	}
	rMsg, err := queries.UpdateMessage(a.Context, param)
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		return res, err
	}
	res.StatusCode = http.StatusOK
	res.ResponseMsg = "Update successful"
	res.Data = rMsg
	return res, err
}

func (a *APIForFrontend) DeleteMessage(id uuid.UUID) (res APIResponse, err error) {
	queries := data.New(a.Tx)
	rMsg, err := queries.DeleteMessage(a.Context, id)
	if err != nil {
		res.StatusCode = http.StatusBadRequest
		return res, err
	}
	res.StatusCode = http.StatusOK
	res.ResponseMsg = "Delete successful"
	res.Data = rMsg
	return res, err
}

func (a *APIForFrontend) ExtendMessageInactiveAt(id uuid.UUID, secret string) (res APIResponse, err error) {
	newSecret, err := secure.GenerateRandomString(ExtensionSecretLength)
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		return res, err
	}
	queries := data.New(a.Tx)
	rMsg, err := queries.UpdateMessageExtendsInactiveAt(a.Context, data.UpdateMessageExtendsInactiveAtParams{
		ExtensionSecret:   newSecret,
		ID:                id,
		ExtensionSecret_2: secret,
	})
	if err != nil {
		res.StatusCode = http.StatusUnauthorized
		return res, err
	}
	res.StatusCode = http.StatusOK
	res.ResponseMsg = "Extend successful"
	res.Data = rMsg
	return res, err
}
