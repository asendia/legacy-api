package api

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/secure"
	"github.com/asendia/legacy-api/simple"
	"github.com/jackc/pgx/v4"
)

func TestMain(m *testing.M) {
	simple.MustLoadEnv("../.env-test.yaml")
	ctx := context.Background()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig("test"))
	if err != nil {
		log.Fatalf("Cannot connect to DB: %v", err)
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		conn.Close()
		log.Fatalf("Cannot begin transaction: %v", err)
	}
	if err := deleteAndCreateTableMessages(ctx, tx); err != nil {
		conn.Close()
		log.Fatalf("Cannot create table messages: %v", err)
	}
	tx.Commit(ctx)
	conn.Close()
	code := m.Run()
	os.Exit(code)
}

func deleteAndCreateTableMessages(ctx context.Context, tx pgx.Tx) error {
	// Delete the table "messages if any"
	qDropTable := "DROP TABLE IF EXISTS public.messages;"
	if _, err := tx.Exec(ctx, string(qDropTable)); err != nil {
		return err
	}
	// Create the table "messages"
	qCreateTable, err := ioutil.ReadFile("../data/schema.sql")
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, string(qCreateTable)); err != nil {
		return err
	}
	return nil
}

func insertTemplateMessage(ctx context.Context, tx pgx.Tx, msgDefault *MessageData) (row data.InsertMessageRow, err error) {
	var msg *MessageData
	if msgDefault == nil {
		msg = generateMessageTemplate()
	} else {
		msg = msgDefault
	}
	encrypted, err := EncryptMessageContent(msg.MessageContent, os.Getenv("ENCRYPTION_KEY"))
	if err != nil {
		return data.InsertMessageRow{}, errors.New(fmt.Sprintf("Encryption failed: %v\n", err))
	}
	queries := data.New(tx)
	row, err = queries.InsertMessage(ctx, data.InsertMessageParams{
		EmailCreator:         msg.EmailCreator,
		EmailReceivers:       msg.EmailReceivers,
		MessageContent:       encrypted,
		InactivePeriodDays:   msg.InactivePeriodDays,
		ReminderIntervalDays: msg.ReminderIntervalDays,
		IsActive:             msg.IsActive,
		ExtensionSecret:      msg.ExtensionSecret,
		InactiveAt:           msg.InactiveAt,
		NextReminderAt:       msg.NextReminderAt,
	})
	if err != nil {
		return row, err
	}
	if row.EmailCreator != msg.EmailCreator {
		return row, errors.New("Inserted data do not match")
	}
	return row, err
}

func generateMessageTemplate() *MessageData {
	rdstr, _ := secure.GenerateRandomString(10)
	extensionSecret, _ := secure.GenerateRandomString(ExtensionSecretLength)
	return &MessageData{
		InactivePeriodDays:   90,
		ReminderIntervalDays: 30,
		MessageContent:       "Hello World!!! " + rdstr,
		EmailCreator:         rdstr + "-asendiamayco@gmail.com",
		EmailReceivers:       []string{"asendia@icloud.com", "crossguard007@yahoo.co.id"},
		ExtensionSecret:      extensionSecret,
	}
}
