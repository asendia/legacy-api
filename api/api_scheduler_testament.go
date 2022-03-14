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
	messageContentMap := map[uuid.UUID]string{}
	for _, row := range rows {
		msgContent := messageContentMap[row.MsgID]
		if msgContent == "" {
			dMsgContent, err := DecryptMessageContent(row.MsgContentEncrypted, os.Getenv("ENCRYPTION_KEY"))
			if err != nil {
				fmt.Printf("Failed to decrypt message: %v\n", err)
				continue
			}
			messageContentMap[row.MsgID] = dMsgContent
			msgContent = dMsgContent
		}
		var howToDecrypt = ""
		if isProbablyClientEncrypted(msgContent) {
			howToDecrypt = "This message is appeared to be client encrypted, you should be able to decrypt it by copy-pasting " +
				`the text begins with "` + encryptPrefixText + `" to https://sejiwo.com, clicking "CLIENT-AES" button and enter the ` +
				"secret text that should have been given to you by the writer of this will." +
				``
		}
		msgParam := mail.TestamentEmailParams{
			Title:                 "Message from " + row.MsgEmailCreator + " sent by sejiwo.com",
			FullName:              row.RcvEmailReceiver,
			EmailCreator:          row.MsgEmailCreator,
			MessageContentPerLine: strings.Split(msgContent, "\n"),
			UnsubscribeURL:        fmt.Sprintf("https://sejiwo.com/?action=unsubscribe-message&id=%s&secret=%s", row.MsgID, row.RcvUnsubscribeSecret),
			HowToDecrypt:          howToDecrypt,
		}
		mmsgHTML, err := mail.GenerateTestamentEmail(msgParam)
		if err != nil {
			fmt.Printf("Failed generating testament email: %v\n", err)
			continue
		}
		mailItems = append(mailItems, mail.MailItem{
			From: mail.MailAddress{
				Email: "noreply@sejiwo.com",
				Name:  "Sejiwo Service",
			},
			To: []mail.MailAddress{
				{
					Email: row.RcvEmailReceiver,
					Name:  "Sejiwo User",
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
	smResList := mail.SendEmails(mailItems)
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
		err := queries.UpdateEmail(a.Context, data.UpdateEmailParams{
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
