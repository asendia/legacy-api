package api

import (
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
	if err = a.queueTelegram("final"); err != nil {
		return res, err
	}
	queries := data.New(a.Tx)
	rows, err := queries.SelectInactiveMessages(a.Context)
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		res.ResponseMsg = "Failed to select inactive messages"
		return
	}
	mailItems := []mail.MailItem{}
	sentRows := []data.SelectInactiveMessagesRow{}
	receiptsEnabled := os.Getenv("TELEGRAM_ENABLED") == "true"
	expected := map[uuid.UUID]data.SelectInactiveMessagesRow{}
	for _, row := range rows {
		expected[row.MsgID] = row
	}
	messageContentMap := map[uuid.UUID]string{}
	for _, row := range rows {
		if receiptsEnabled {
			var sent bool
			if err := a.Tx.QueryRow(a.Context, `SELECT EXISTS(SELECT 1 FROM email_delivery_receipts WHERE message_id=$1 AND email_receiver=$2 AND extension_secret=$3 AND cycle_date=$4)`, row.MsgID, row.RcvEmailReceiver, row.MsgExtensionSecret, row.MsgInactiveAt).Scan(&sent); err != nil {
				return res, err
			}
			if sent {
				continue
			}
		}
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
			UnsubscribeURL:        fmt.Sprintf("https://sejiwo.com/unsubscribe?id=%s&secret=%s", row.MsgID, row.RcvUnsubscribeSecret),
			HowToDecrypt:          howToDecrypt,
		}
		mmsgHTML, err := mail.GenerateTestamentEmail(msgParam)
		if err != nil {
			fmt.Printf("Failed generating testament email: %v\n", err)
			continue
		}
		sentRows = append(sentRows, row)
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
	smResList := a.sendEmails(mailItems)
	success := map[uuid.UUID]int{}
	for index, result := range smResList {
		if index >= len(sentRows) {
			break
		}
		row := sentRows[index]
		if result.Err != nil {
			continue
		}
		success[row.MsgID]++
		if receiptsEnabled {
			if _, err := a.Tx.Exec(a.Context, `INSERT INTO email_delivery_receipts(message_id,email_receiver,extension_secret,cycle_date) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, row.MsgID, row.RcvEmailReceiver, row.MsgExtensionSecret, row.MsgInactiveAt); err != nil {
				return res, err
			}
		}
	}
	// Advance each message once. Stored receipts protect partial retries when Telegram is enabled.
	for id, row := range expected {
		if receiptsEnabled {
			var pending int
			if err := a.Tx.QueryRow(a.Context, `SELECT count(*) FROM messages_email_receivers r WHERE r.message_id=$1 AND NOT r.is_unsubscribed AND NOT EXISTS(SELECT 1 FROM email_delivery_receipts e WHERE e.message_id=r.message_id AND e.email_receiver=r.email_receiver AND e.extension_secret=$2 AND e.cycle_date=$3)`, id, row.MsgExtensionSecret, row.MsgInactiveAt).Scan(&pending); err != nil {
				return res, err
			}
			if pending > 0 {
				continue
			}
		} else if success[id] == 0 {
			continue
		}
		if _, updateError := queries.UpdateMessageAfterSendingTestament(a.Context, id); updateError != nil {
			return res, updateError
		}
	}

	res.StatusCode = http.StatusOK
	res.ResponseMsg = "Testament emails sent successfully"
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
	res.Data = rows
	return res, err
}
