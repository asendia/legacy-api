package p

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/asendia/legacy-api/api"
	"github.com/asendia/legacy-api/data"
	"github.com/joho/godotenv"
)

// Google Cloud Function
func CloudFunctionForSchedulerWithStaticSecret(ctx context.Context, m PubSubMessage) error {
	godotenv.Load()
	// Establishing connection to database
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig(os.Getenv("ENVIRONMENT")))
	if err != nil {
		log.Printf("Cannot connect to the database: %v\n", err.Error())
		return err
	}
	defer conn.Close()
	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Printf("Cannot begin database transaction: %v\n", err.Error())
		return err
	}
	defer tx.Rollback(ctx)
	a := api.APIForScheduler{
		Context: ctx,
		Tx:      tx,
	}
	var res api.APIResponse
	action := m.Attributes["action"]
	switch action {
	case "send-reminder-messages":
		res, err = a.SendReminderMessages()
		break
	case "send-testaments":
		res, err = a.SendTestamentsOfInactiveMessages()
		break
	case "select-messages-need-reminding":
		res, err = a.SelectMessagesNeedReminding()
		break
	case "select-inactive-messages":
		res, err = a.SelectInactiveMessages()
		break
	default:
		err = errors.New("Invalid Action")
		res.StatusCode = http.StatusNotFound
		break
	}
	// Handle controller error
	if err != nil {
		log.Printf("Controller error: %+v\n", err)
		return err
	}
	// Generate response
	resStr, err := res.ToString()
	if err != nil {
		log.Printf("Cannot generate a response: %v\n", err)
		return err
	}
	tx.Commit(ctx)
	log.Printf("Success action: %s, response: %s", action, resStr)
	return nil
}

type PubSubRequest struct {
	Message      PubSubMessage `json:"message"`
	Subscription string        `json:"subscription"`
}

type PubSubMessage struct {
	Data       []byte            `json:"data"`
	Attributes map[string]string `json:"attributes"`
}

func VerifySecretHeader(r *http.Request) (statusCode int, err error) {
	serverSecret := os.Getenv("STATIC_SECRET")
	if len(serverSecret) != api.ExtensionSecretLength {
		return http.StatusInternalServerError, errors.New("env STATIC_SECRET is invalid: " + serverSecret)
	}
	clientSecret := r.Header.Get("x-static-secret")
	if serverSecret != clientSecret {
		return http.StatusUnauthorized, errors.New("Invalid secret header: " + clientSecret)
	}
	return http.StatusOK, nil
}
