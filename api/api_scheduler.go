package api

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type APIForScheduler struct {
	Context context.Context
	Tx      pgx.Tx
}
