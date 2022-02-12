package api

import (
	"net/http"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/secure"
	"github.com/google/uuid"
)

func (a *APIForFrontend) DeleteMessage(jwtRes secure.JWTResponse, id uuid.UUID) (res APIResponse, err error) {
	queries := data.New(a.Tx)
	row, err := queries.DeleteMessage(a.Context, data.DeleteMessageParams{
		ID:           id,
		EmailCreator: jwtRes.Email,
	})
	if err != nil {
		res.StatusCode = http.StatusBadRequest
		return res, err
	}
	res.StatusCode = http.StatusOK
	res.ResponseMsg = "Delete successful"
	res.Data = MessageData{
		ID:                   row.ID,
		CreatedAt:            row.CreatedAt,
		EmailCreator:         row.EmailCreator,
		InactivePeriodDays:   row.InactivePeriodDays,
		ReminderIntervalDays: row.ReminderIntervalDays,
		IsActive:             row.IsActive,
		ExtensionSecret:      row.ExtensionSecret,
		InactiveAt:           row.InactiveAt,
		NextReminderAt:       row.NextReminderAt,
	}
	return res, err
}
