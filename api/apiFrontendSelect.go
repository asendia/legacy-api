package api

import (
	"net/http"
	"os"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/mail"
	"github.com/asendia/legacy-api/secure"
	"github.com/google/uuid"
)

func (a *APIForFrontend) SelectMessagesByEmailCreator(jwtRes secure.JWTResponse) (res APIResponse, err error) {
	emailCreator := jwtRes.Email
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
	msgs := []*MessageData{}
	currentMessageID := uuid.UUID{}
	for _, row := range rows {
		var msg *MessageData
		if row.MsgID != currentMessageID {
			msgContent, err := DecryptMessageContent(row.MsgContentEncrypted, os.Getenv("ENCRYPTION_KEY"))
			if err != nil {
				return res, err
			}
			msg = &MessageData{
				ID:                   row.MsgID,
				CreatedAt:            row.MsgCreatedAt,
				EmailCreator:         row.MsgEmailCreator,
				EmailReceivers:       []string{},
				MessageContent:       msgContent,
				InactivePeriodDays:   row.MsgInactivePeriodDays,
				ReminderIntervalDays: row.MsgReminderIntervalDays,
				IsActive:             row.MsgIsActive,
				ExtensionSecret:      row.MsgExtensionSecret,
				InactiveAt:           row.MsgInactiveAt,
				NextReminderAt:       row.MsgNextReminderAt,
			}
			msgs = append(msgs, msg)
		} else {
			msg = msgs[len(msgs)-1]
		}
		msg.EmailReceivers = append(msg.EmailReceivers, row.RcvEmailReceiver)
		currentMessageID = row.MsgID
	}
	res.Data = msgs
	res.StatusCode = http.StatusOK
	res.ResponseMsg = "Select messages successful"
	return res, err
}
