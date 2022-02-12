package api

import (
	"net/http"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/secure"
	"github.com/google/uuid"
)

func (a *APIForFrontend) ExtendMessageInactiveAt(secret string, id uuid.UUID) (res APIResponse, err error) {
	newSecret, err := secure.GenerateRandomString(ExtensionSecretLength)
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		return res, err
	}
	queries := data.New(a.Tx)
	row, err := queries.UpdateMessageExtendsInactiveAt(a.Context, data.UpdateMessageExtendsInactiveAtParams{
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
	res.Data = MessageData{
		ID:                   row.ID,
		CreatedAt:            row.CreatedAt,
		EmailCreator:         row.EmailCreator,
		InactivePeriodDays:   row.InactivePeriodDays,
		ReminderIntervalDays: row.ReminderIntervalDays,
		IsActive:             row.IsActive,
		ExtensionSecret:      row.ExtensionSecret,
		InactiveAt:           row.InactiveAt,
		NextReminderAt:       row.NextReminderAt,
	}
	return res, err
}

func (a *APIForFrontend) UnsubscribeMessage(secret string, messageID uuid.UUID) (res APIResponse, err error) {
	queries := data.New(a.Tx)
	msgRcvr, err := queries.UpdateMessagesEmailReceiverUnsubscribe(a.Context, data.UpdateMessagesEmailReceiverUnsubscribeParams{
		MessageID:         messageID,
		UnsubscribeSecret: secret,
	})
	if err != nil {
		res.StatusCode = http.StatusForbidden
		return res, err
	}
	res.StatusCode = http.StatusOK
	res.ResponseMsg = "Unsubscribe successful: " + msgRcvr.EmailReceiver
	return res, err
}
