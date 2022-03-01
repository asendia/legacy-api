package api

import (
	"net/http"
	"os"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/secure"
)

func (a *APIForFrontend) UpdateMessage(jwtRes secure.JWTResponse, param APIParamUpdateMessage) (res APIResponse, err error) {
	queries := data.New(a.Tx)
	// Refresh extension secret on every update
	extensionSecret, err := secure.GenerateRandomString(ExtensionSecretLength)
	if err != nil {
		return res, err
	}
	contentEncrypted, err := EncryptMessageContent(param.MessageContent, os.Getenv("ENCRYPTION_KEY"))
	if err != nil {
		return res, err
	}
	row, err := queries.UpdateMessage(a.Context, data.UpdateMessageParams{
		ContentEncrypted:     contentEncrypted,
		InactivePeriodDays:   param.InactivePeriodDays,
		ReminderIntervalDays: param.ReminderIntervalDays,
		IsActive:             param.IsActive,
		ExtensionSecret:      extensionSecret,
		ID:                   param.ID,
		EmailCreator:         jwtRes.Email,
	})
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		return res, err
	}
	rReceivers, err := queries.SelectMessagesEmailReceiversNotUnsubscribed(a.Context, param.ID)
	if err != nil {
		return res, err
	}
	mEmail := diffOldWithNewEmailList(rReceivers, param.EmailReceivers)
	newReceivers := []string{}
	for email, action := range mEmail {
		if action == "insert" {
			unsubscribeSecret, err := secure.GenerateRandomString(ExtensionSecretLength)
			if err != nil {
				return res, err
			}
			err = queries.InsertEmailIgnoreConflict(a.Context, email)
			if err != nil {
				return res, err
			}
			_, err = queries.InsertMessagesEmailReceiver(a.Context, data.InsertMessagesEmailReceiverParams{
				MessageID:         param.ID,
				EmailReceiver:     email,
				UnsubscribeSecret: unsubscribeSecret,
			})
			if err != nil {
				return res, err
			}
			newReceivers = append(newReceivers, email)
		} else if action == "delete" {
			err = queries.DeleteMessagesEmailReceiver(a.Context, data.DeleteMessagesEmailReceiverParams{
				MessageID:     param.ID,
				EmailReceiver: email,
			})
		} else if action == "ignore" {
			newReceivers = append(newReceivers, email)
		}
	}

	res.StatusCode = http.StatusOK
	res.ResponseMsg = "Update successful"
	res.Data = MessageData{
		ID:                   row.ID,
		CreatedAt:            row.CreatedAt,
		EmailCreator:         row.EmailCreator,
		EmailReceivers:       newReceivers,
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

func diffOldWithNewEmailList(oldList []data.MessagesEmailReceiver, newList []string) (actionMap map[string]string) {
	actionMap = map[string]string{}
	for _, email := range newList {
		actionMap[email] = "insert"
	}
	for _, rcv := range oldList {
		action := ""
		if rcv.IsUnsubscribed {
			action = "hide"
		} else if actionMap[rcv.EmailReceiver] == "insert" {
			action = "ignore"
		} else if actionMap[rcv.EmailReceiver] == "" {
			action = "delete"
		}
		actionMap[rcv.EmailReceiver] = action
	}
	return actionMap
}
