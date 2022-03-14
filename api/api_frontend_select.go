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
	msgMap := map[uuid.UUID]*MessageData{}
	msgs := []*MessageData{}
	for _, row := range rows {
		if msgMap[row.MsgID] == nil {
			msgContent, err := DecryptMessageContent(row.MsgContentEncrypted, os.Getenv("ENCRYPTION_KEY"))
			if err != nil {
				return res, err
			}
			msgMap[row.MsgID] = &MessageData{
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
				NextReminderAt:       row.MsgNextReminderAt}
			msgs = append(msgs, msgMap[row.MsgID])
		}
		if row.RcvEmailReceiver.Valid && row.RcvIsUnsubscribed.Valid && !row.RcvIsUnsubscribed.Bool {
			msgMap[row.MsgID].EmailReceivers = append(msgMap[row.MsgID].EmailReceivers, row.RcvEmailReceiver.String)
		}
	}
	res.Data = msgs
	res.StatusCode = http.StatusOK
	res.ResponseMsg = "Select messages successful"
	return res, err
}
