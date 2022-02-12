package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

type APIForFrontend struct {
	Context context.Context
	Tx      pgx.Tx
}

type APIParamInsertMessage struct {
	EmailReceivers       []string
	MessageContent       string
	InactivePeriodDays   int32
	ReminderIntervalDays int32
}

func ParseReqInsertMessage(r *http.Request) (p APIParamInsertMessage, err error) {
	err = json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		return
	}
	err = validateEmails(p.EmailReceivers)
	if err != nil {
		return
	}
	err = validateInactivePeriodDays(p.InactivePeriodDays)
	if err != nil {
		return
	}
	err = validateReminderIntervalDays(p.ReminderIntervalDays)
	if err != nil {
		return
	}
	err = validateMessageContent(p.MessageContent)
	return
}

type APIParamSelectMessagesByEmailCreator struct {
	EmailCreator string
}

func ParseReqSelectMessagesByEmailCreator(r *http.Request) (p APIParamSelectMessagesByEmailCreator, err error) {
	err = json.NewDecoder(r.Body).Decode(&p)
	return
}

type APIParamUpdateMessage struct {
	MessageContent       string
	InactivePeriodDays   int32
	ReminderIntervalDays int32
	IsActive             bool
	ExtensionSecret      string
	ID                   uuid.UUID
	EmailReceivers       []string
}

func ParseReqUpdateMessage(r *http.Request) (p APIParamUpdateMessage, err error) {
	err = json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		return
	}
	err = validateEmails(p.EmailReceivers)
	if err != nil {
		return
	}
	err = validateInactivePeriodDays(p.InactivePeriodDays)
	if err != nil {
		return
	}
	err = validateReminderIntervalDays(p.ReminderIntervalDays)
	if err != nil {
		return
	}
	err = validateMessageContent(p.MessageContent)
	return
}

type APIParamDeleteMessageByID struct {
	ID uuid.UUID
}

func ParseReqDeleteMessage(r *http.Request) (p APIParamDeleteMessageByID, err error) {
	err = json.NewDecoder(r.Body).Decode(&p)
	return
}

func validateEmails(emails []string) error {
	_, err := mail.ParseAddressList(strings.Join(emails, ","))
	return err
}

func validateInactivePeriodDays(days int32) error {
	if days < 90 || days > 360 {
		return errors.New("InactivePeriodDays should be set to 90 to 360 days")
	}
	return nil
}

func validateReminderIntervalDays(days int32) error {
	if days < 15 || days > 30 {
		return errors.New("ReminderIntervalDays should be set to 15 to 30 days")
	}
	return nil
}

func validateMessageContent(cnt string) error {
	if len(cnt) < 10 || len(cnt) > 800 {
		return errors.New("MessageContent length should be between 10 to 800 characters")
	}
	return nil
}
