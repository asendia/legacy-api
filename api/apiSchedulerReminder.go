package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/mail"
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
	currentEmailCreator := ""
	for _, row := range rows {
		if row.MsgEmailCreator != currentEmailCreator {
			msgs = append(msgs, &MessageData{
				ID:                   row.MsgID,
				CreatedAt:            row.MsgCreatedAt,
				EmailCreator:         row.MsgEmailCreator,
				EmailReceivers:       []string{row.RcvEmailReceiver},
				InactivePeriodDays:   row.MsgInactivePeriodDays,
				ReminderIntervalDays: row.MsgReminderIntervalDays,
				IsActive:             row.MsgIsActive,
				ExtensionSecret:      row.MsgExtensionSecret,
				InactiveAt:           row.MsgInactiveAt,
				NextReminderAt:       row.MsgNextReminderAt,
			})
			currentEmailCreator = row.MsgEmailCreator
		} else {
			msg := msgs[len(msgs)-1]
			msg.EmailReceivers = append(msg.EmailReceivers, row.RcvEmailReceiver)
		}
	}
	for _, msg := range msgs {
		param := mail.ReminderEmailParams{
			Title:          "Reminder to extend your warisin.com message",
			FullName:       "Warisin User",
			InactiveAt:     msg.InactiveAt.Local().Format("2006-01-02"),
			EmailReceivers: msg.EmailReceivers,
			ExtensionURL:   fmt.Sprintf("https://warisin.com/?action=extend-message&id=%s&secret=%s", msg.ID, msg.ExtensionSecret),
		}
		htmlContent, err := mail.GenerateReminderEmail(param)
		if err != nil {
			fmt.Printf("Cannot generate reminder email: %v\n", err)
			continue
		}
		mail := mail.MailItem{
			From: mail.MailAddress{
				Email: "noreply@warisin.com",
				Name:  "Warisin Service",
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
	eClient := mail.Mailjet{PublicKey: os.Getenv("MAILJET_PUBLIC_KEY"), PrivateKey: os.Getenv("MAILJET_PRIVATE_KEY")}
	smResList, err := eClient.SendEmails(mailItems)
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		res.ResponseMsg = "Failed to send reminder emails: " + err.Error()
		return res, err
	}
	for id, smRes := range smResList {
		if smRes.Err == nil {
			_, err := queries.UpdateMessageAfterSendingReminder(a.Context, msgs[id].ID)
			if err != nil {
				fmt.Printf("Failed to update message inactive_at and next_reminder_at: %v\n", err)
				smRes.Err = err
			}
			continue
		}
		fmt.Printf("An email probably gets an error: %v\n", smRes.Err)
		_, err := queries.UpdateEmail(a.Context, data.UpdateEmailParams{
			IsActive: false,
			Email:    msgs[id].EmailCreator,
		})
		if err != nil {
			fmt.Printf("Failed updating email IsActive status: %v\n", err.Error())
			smRes.Err = errors.New(smRes.Err.Error() + " & " + err.Error())
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
