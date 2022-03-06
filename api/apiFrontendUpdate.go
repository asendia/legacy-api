package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/secure"
	"github.com/google/uuid"
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
	messageIDs := []uuid.UUID{}
	unsubscribeSecrets := []string{}
	is_unsubscribeds := []bool{}
	for i := 0; i < len(param.EmailReceivers); i++ {
		messageIDs = append(messageIDs, param.ID)
		is_unsubscribeds = append(is_unsubscribeds, false)
		unsubscribeSecret, err := secure.GenerateRandomString(ExtensionSecretLength)
		if err != nil {
			return res, err
		}
		unsubscribeSecrets = append(unsubscribeSecrets, unsubscribeSecret)
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
		fmt.Printf("Failed to UpdateMessage: %v", err)
		res.StatusCode = http.StatusInternalServerError
		return res, err
	}
	receiverRows, err := queries.UpsertReceivers(a.Context, data.UpsertReceiversParams{
		MessageID:          row.ID,
		EmailReceivers:     param.EmailReceivers,
		UnsubscribeSecrets: unsubscribeSecrets,
	})
	if err != nil {
		fmt.Printf("Failed to UpsertReceivers: %v", err)
		res.StatusCode = http.StatusInternalServerError
		return res, err
	}
	receivers := []string{}
	for _, r := range receiverRows {
		receivers = append(receivers, r.EmailReceiver)
	}
	res.StatusCode = http.StatusOK
	res.ResponseMsg = "Update successful"
	res.Data = MessageData{
		ID:                   row.ID,
		CreatedAt:            row.CreatedAt,
		EmailCreator:         row.EmailCreator,
		EmailReceivers:       param.EmailReceivers,
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
