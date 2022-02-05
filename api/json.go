package api

import (
	"errors"
	"strings"
	"time"

	"github.com/asendia/legacy-api/secure"
	"github.com/google/uuid"
)

const ExtensionSecretLength = 69

type MessageData struct {
	ID                   uuid.UUID `json:"id"`
	CreatedAt            time.Time `json:"createdAt"`
	EmailCreator         string    `json:"emailCreator"`
	EmailReceivers       []string  `json:"emailReceivers"`
	MessageContent       string    `json:"messageContent"`
	InactivePeriodDays   int32     `json:"inactivePeriodDays"`
	ReminderIntervalDays int32     `json:"reminderIntervalDays"`
	IsActive             bool      `json:"isActive"`
	ExtensionSecret      string    `json:"extension_secret"`
	InactiveAt           time.Time `json:"inactiveAt"`
	NextReminderAt       time.Time `json:"nextReminderAt"`
}

type SchedulerData struct {
	ID     int32  `json:"id"`
	Secret string `json:"secret"`
}

func DecryptMessageContent(str string, secret string) (string, error) {
	encryptedArr := strings.Split(str, ".")
	if len(encryptedArr) != 2 {
		return "", errors.New("Invalid encrypted string")
	}
	msgContent, err := secure.Decrypt(
		secure.EncryptResult{IV: encryptedArr[0], Text: encryptedArr[1]},
		secret)
	return msgContent, err
}

func EncryptMessageContent(str string, secret string) (string, error) {
	encrypted, err := secure.Encrypt(str, secret)
	if err != nil {
		return "", err
	}
	return encrypted.IV + "." + encrypted.Text, nil
}
