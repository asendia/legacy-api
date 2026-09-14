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
	queries := data.New(a.Tx)
	receiptsEnabled := os.Getenv("TELEGRAM_ENABLED") == "true"
	var rows []data.SelectInactiveMessagesRow
	if receiptsEnabled {
		rows, err = queries.SelectPendingEmails(a.Context)
	} else {
		rows, err = queries.SelectInactiveMessages(a.Context)
	}
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		res.ResponseMsg = "Failed to select inactive messages"
		return
	}
	if err = a.queueTelegramFinal(rows); err != nil {
		return res, err
	}
	// Record each selected attempt, including local preparation failures.
	if receiptsEnabled {
		for _, row := range rows {
			if _, err = a.Tx.Exec(a.Context, `INSERT INTO email_delivery_receipts
                (message_id,email_receiver,extension_secret,cycle_date,sent_at,attempts,next_attempt_at,stopped_at)
                VALUES($1,$2,$3,$4,NULL,1,now()+interval '1 hour',NULL)
                ON CONFLICT(message_id,email_receiver,extension_secret,cycle_date) DO UPDATE
                SET attempts=email_delivery_receipts.attempts+1,next_attempt_at=now()+interval '1 hour',
                stopped_at=CASE WHEN email_delivery_receipts.attempts+1>=10 THEN now() ELSE NULL END`,
				row.MsgID, row.RcvEmailReceiver, row.MsgExtensionSecret, row.MsgInactiveAt); err != nil {
				return res, err
			}
		}
	}
	mailItems := []mail.MailItem{}
	sentRows := []data.SelectInactiveMessagesRow{}
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
			if _, err := a.Tx.Exec(a.Context, `UPDATE email_delivery_receipts SET sent_at=now(),stopped_at=NULL WHERE message_id=$1 AND email_receiver=$2 AND extension_secret=$3 AND cycle_date=$4`, row.MsgID, row.RcvEmailReceiver, row.MsgExtensionSecret, row.MsgInactiveAt); err != nil {
				return res, err
			}
		}
	}
	// Completion must also find cycles with no remaining send rows.
	var completed []uuid.UUID
	if receiptsEnabled {
		completed, err = queries.SelectCompletedEmailCycles(a.Context)
		if err != nil {
			return res, err
		}
	} else {
		for id := range success {
			completed = append(completed, id)
		}
	}
	for _, id := range completed {
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
