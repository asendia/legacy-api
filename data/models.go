// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.15.0

package data

import (
	"time"

	"github.com/google/uuid"
)

type Email struct {
	Email     string
	CreatedAt time.Time
	IsActive  bool
}

type Message struct {
	ID                   uuid.UUID
	EmailCreator         string
	CreatedAt            time.Time
	ContentEncrypted     string
	InactivePeriodDays   int32
	ReminderIntervalDays int32
	IsActive             bool
	ExtensionSecret      string
	InactiveAt           time.Time
	NextReminderAt       time.Time
	SentCounter          int32
}

type MessagesEmailReceiver struct {
	MessageID         uuid.UUID
	EmailReceiver     string
	IsUnsubscribed    bool
	UnsubscribeSecret string
}
