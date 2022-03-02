package api

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/secure"
	"github.com/asendia/legacy-api/simple"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var pgxPoolConn *pgxpool.Pool

func TestMain(m *testing.M) {
	simple.MustLoadEnv("../.env-test.yaml")
	ctx := context.Background()
	var err error
	pgxPoolConn, err = data.ConnectDB(ctx, data.LoadDBURLConfig())
	if err != nil {
		log.Fatalf("Cannot connect to DB: %+v %+v\n", pgxPoolConn, err)
		return
	}
	tx, err := pgxPoolConn.Begin(ctx)
	if err != nil {
		log.Fatalf("Cannot begin transaction: %v\n", err)
		return
	}
	if err := deleteAndCreateTableMessages(ctx, tx); err != nil {
		log.Fatalf("Cannot create table messages: %v\n", err)
		return
	}
	tx.Commit(ctx)
	code := m.Run()
	pgxPoolConn.Close()
	os.Exit(code)
}

func deleteAndCreateTableMessages(ctx context.Context, tx pgx.Tx) error {
	// Delete the table "messages if any"
	qDropTable := `DROP TABLE IF EXISTS public.messages_email_receivers;
	DROP TABLE IF EXISTS public.messages;
	DROP TABLE IF EXISTS public.emails;
	`
	if _, err := tx.Exec(ctx, string(qDropTable)); err != nil {
		return err
	}
	// Create the tables
	qCreateTable, err := ioutil.ReadFile("../data/schema.sql")
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, string(qCreateTable)); err != nil {
		return err
	}
	// Create the tables
	qGrantRole, err := ioutil.ReadFile("../data/schema_test.sql")
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, string(qGrantRole)); err != nil {
		return err
	}
	return nil
}

func generateMessageTemplate() MessageData {
	rdstr, _ := secure.GenerateRandomString(10)
	extensionSecret, _ := secure.GenerateRandomString(ExtensionSecretLength)
	return MessageData{
		InactivePeriodDays:   90,
		ReminderIntervalDays: 15,
		MessageContent:       "Hello World!!! " + rdstr,
		EmailCreator:         rdstr + "-asendiamayco@gmail.com",
		EmailReceivers:       []string{rdstr + "-asendia@icloud.com", rdstr + "-crossguard007@yahoo.co.id"},
		ExtensionSecret:      extensionSecret,
		IsActive:             true,
	}
}

func generateJwtMessageTemplate(email string) secure.JWTResponse {
	return secure.JWTResponse{
		Email: email,
	}
}
