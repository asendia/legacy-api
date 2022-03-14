package api

import (
	"fmt"
	"net/http"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/mail"
	"github.com/google/uuid"
)

func (a *APIForScheduler) SendReminderMessages() (res APIResponse, err error) {
	queries := data.New(a.Tx)
	rows, err := queries.SelectMessagesNeedReminding(a.Context)
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		res.ResponseMsg = "Failed to select messages need reminding"
		return res, err
	}
	mailItems := []mail.MailItem{}
	msgs := []*MessageData{}
	msgMap := map[uuid.UUID]*MessageData{}
	for _, row := range rows {
		if msgMap[row.MsgID] == nil {
			msgMap[row.MsgID] = &MessageData{
				ID:                   row.MsgID,
				CreatedAt:            row.MsgCreatedAt,
				EmailCreator:         row.MsgEmailCreator,
				EmailReceivers:       []string{},
				InactivePeriodDays:   row.MsgInactivePeriodDays,
				ReminderIntervalDays: row.MsgReminderIntervalDays,
				IsActive:             row.MsgIsActive,
				ExtensionSecret:      row.MsgExtensionSecret,
				InactiveAt:           row.MsgInactiveAt,
				NextReminderAt:       row.MsgNextReminderAt,
			}
			msgs = append(msgs, msgMap[row.MsgID])
		}
		msgMap[row.MsgID].EmailReceivers = append(msgMap[row.MsgID].EmailReceivers, row.RcvEmailReceiver)
	}
	for _, msg := range msgs {
		param := mail.ReminderEmailParams{
			Title:              "Reminder to extend your sejiwo.com message",
			FullName:           "Sejiwo User",
			InactiveAt:         msg.InactiveAt.Local().Format("2006-01-02"),
			TestamentReceivers: msg.EmailReceivers,
			ExtensionURL:       fmt.Sprintf("https://sejiwo.com/?action=extend-message&id=%s&secret=%s", msg.ID, msg.ExtensionSecret),
		}
		htmlContent, err := mail.GenerateReminderEmail(param)
		if err != nil {
			fmt.Printf("Cannot generate reminder email: %v\n", err)
			continue
		}
		mail := mail.MailItem{
			From: mail.MailAddress{
				Email: "noreply@sejiwo.com",
				Name:  "Sejiwo Service",
			},
			To: []mail.MailAddress{
				{
					Email: msg.EmailCreator,
					Name:  param.FullName,
				},
			},
			Subject:     param.Title,
			HtmlContent: htmlContent,
		}
		mailItems = append(mailItems, mail)
	}
	if len(mailItems) == 0 {
		res.StatusCode = http.StatusOK
		res.ResponseMsg = "No reminder message is sent this time"
		return
	}
	smResList := mail.SendEmails(mailItems)
	for id, smRes := range smResList {
		if smRes.Err == nil {
			_, err := queries.UpdateMessageAfterSendingReminder(a.Context, msgs[id].ID)
			if err != nil {
				fmt.Printf("Failed to update message inactive_at and next_reminder_at: %v\n", err)
				smRes.Err = err
			}
			continue
		}
	}
	res.StatusCode = http.StatusOK
	res.ResponseMsg = fmt.Sprintf("Reminder emails sent successfully")
	res.Data = smResList
	return res, err
}

func (a *APIForScheduler) SelectMessagesNeedReminding() (res APIResponse, err error) {
	queries := data.New(a.Tx)
	rows, err := queries.SelectMessagesNeedReminding(a.Context)
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		return res, err
	}
	msgs := []MessageData{}
	for _, row := range rows {
		msgs = append(msgs, MessageData{
			ID:                   row.MsgID,
			CreatedAt:            row.MsgCreatedAt,
			EmailCreator:         row.MsgEmailCreator,
			InactivePeriodDays:   row.MsgInactivePeriodDays,
			ReminderIntervalDays: row.MsgReminderIntervalDays,
			IsActive:             row.MsgIsActive,
			ExtensionSecret:      row.MsgExtensionSecret,
			InactiveAt:           row.MsgInactiveAt,
			NextReminderAt:       row.MsgNextReminderAt,
		})
	}
	res.Data = msgs
	return res, err
}
