package api

import (
	"context"

	"github.com/jackc/pgx/v4"
)

type APIForScheduler struct {
	Context context.Context
	Tx      pgx.Tx
}
