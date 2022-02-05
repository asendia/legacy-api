package api

import (
	"context"
	"net/http"
	"os"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/simple"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

type APIForScheduler struct {
	Context context.Context
	Tx      pgx.Tx
}

// Machine facing queries
func (a *APIForScheduler) SelectMessagesNeedReminding() (res APIResponse, err error) {
	queries := data.New(a.Tx)
	res.Data, err = queries.SelectMessagesNeedReminding(a.Context, simple.TimeTodayUTC())
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
		return res, err
	}
	return res, err
}

func (a *APIForScheduler) UpdateMessageAfterSendingReminder(id uuid.UUID) (res APIResponse, err error) {
	queries := data.New(a.Tx)
	res.Data, err = queries.UpdateMessageAfterSendingReminder(a.Context, id)
	if err != nil {
		res.StatusCode = http.StatusNotFound
	}
	return res, err
}

func (a *APIForScheduler) SelectInactiveMessages() (res APIResponse, err error) {
	queries := data.New(a.Tx)
	rows, err := queries.SelectInactiveMessages(a.Context, simple.TimeTodayUTC())
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
	}
	for _, row := range rows {
		row.MessageContent, err = DecryptMessageContent(row.MessageContent, os.Getenv("ENCRYPTION_KEY"))
		if err != nil {
			return res, err
		}
	}
	res.Data = rows
	return res, err
}

func (a *APIForScheduler) UpdateMessageAfterSendingTestament(id uuid.UUID) (res APIResponse, err error) {
	queries := data.New(a.Tx)
	res.Data, err = queries.UpdateMessageAfterSendingTestament(a.Context, id)
	if err != nil {
		res.StatusCode = http.StatusInternalServerError
	}
	return res, err
}
