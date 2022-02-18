package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/mail"
	"github.com/google/uuid"
)

// Machine facing queries
func (a *APIForScheduler) SendTestamentsOfInactiveMessages() (res APIResponse, err error) {
	queries := data.New(a.Tx)
	rows, err := queries.SelectInactiveMessages(a.Context)
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		res.ResponseMsg = "Failed to select inactive messages"
		return
	}
	mailItems := []mail.MailItem{}
	for _, row := range rows {
		msgContent, err := DecryptMessageContent(row.MsgContentEncrypted, os.Getenv("ENCRYPTION_KEY"))
		if err != nil {
			fmt.Printf("Failed to decrypt message: %v\n", err)
			continue
		}
		msgParam := mail.TestamentEmailParams{
			Title:                 "Message from " + row.MsgEmailCreator + " sent by warisin.com",
			FullName:              row.RcvEmailReceiver,
			EmailCreator:          row.MsgEmailCreator,
			MessageContentPerLine: strings.Split(msgContent, "\n"),
			UnsubscribeURL:        fmt.Sprintf("https://warisin.com/?action=unsubscribe-message&id=%s&secret=%s", row.MsgID, row.RcvUnsubscribeSecret),
		}
		mmsgHTML, err := mail.GenerateTestamentEmail(msgParam)
		if err != nil {
			fmt.Printf("Failed generating testament email: %v\n", err)
			continue
		}
		mailItems = append(mailItems, mail.MailItem{
			From: mail.MailAddress{
				Email: "noreply@warisin.com",
				Name:  "Warisin Service",
			},
			To: []mail.MailAddress{
				{
					Email: row.RcvEmailReceiver,
					Name:  "Warisin User",
				},
			},
			Subject:     msgParam.Title,
			HtmlContent: mmsgHTML,
		})
	}
	if len(mailItems) == 0 {
		res.StatusCode = http.StatusOK
		res.ResponseMsg = "No testament message is sent this time"
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
			_, err := queries.UpdateMessageAfterSendingTestament(a.Context, rows[id].MsgID)
			if err != nil {
				fmt.Printf("Failed to update message inactive_at and next_reminder_at: %v\n", err)
				smRes.Err = err
			}
			continue
		}
		fmt.Printf("An email probably gets an error: %v\n", smRes.Err)
		_, err := queries.UpdateEmail(a.Context, data.UpdateEmailParams{
			IsActive: false,
			Email:    rows[id].RcvEmailReceiver,
		})
		if err != nil {
			fmt.Printf("Failed updating email IsActive status: %v\n", err.Error())
			smRes.Err = errors.New(smRes.Err.Error() + " & " + err.Error())
		}
	}
	res.StatusCode = http.StatusOK
	res.ResponseMsg = fmt.Sprintf("Testament emails sent successfully")
	res.Data = smResList
	return res, err
}

func (a *APIForScheduler) SelectInactiveMessages() (res APIResponse, err error) {
	queries := data.New(a.Tx)
	rows, err := queries.SelectInactiveMessages(a.Context)
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		return
	}
	msgs := []MessageData{}
	currentMessageID := uuid.UUID{}
	for _, row := range rows {
		var msg MessageData
		if currentMessageID != row.MsgID {
			msg = MessageData{
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
			}
			msg.MessageContent, err = DecryptMessageContent(row.MsgContentEncrypted, os.Getenv("ENCRYPTION_KEY"))
			if err != nil {
				return res, err
			}
		} else {
			msg = msgs[len(msgs)-1]
			msg.EmailReceivers = append(msg.EmailReceivers, row.RcvEmailReceiver)
		}
		msgs = append(msgs)
	}
	res.Data = rows
	return res, err
}
