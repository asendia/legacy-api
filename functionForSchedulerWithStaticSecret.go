package p

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/asendia/legacy-api/api"
	"github.com/asendia/legacy-api/data"
	"github.com/joho/godotenv"
)

// Google Cloud Function
func CloudFunctionForSchedulerWithStaticSecret(w http.ResponseWriter, r *http.Request) {
	godotenv.Load()
	statusCode, err := VerifySecretHeader(r)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Decode and verify auth failed", statusCode)
		return
	}
	// Establishing connection to database
	ctx := r.Context()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig(os.Getenv("ENVIRONMENT")))
	if err != nil {
		log.Printf("Cannot connect to the database: %v\n", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Printf("Cannot begin database transaction: %v\n", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)
	a := api.APIForScheduler{
		Context: ctx,
		Tx:      tx,
	}
	var res api.APIResponse
	action := r.URL.Query().Get("action")
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
		log.Println(err.Error())
		http.Error(w, err.Error(), res.GetValidStatusCode())
		return
	}
	// Generate response
	resStr, err := res.ToString()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Cannot generate a response", http.StatusInternalServerError)
		return
	}
	tx.Commit(ctx)
	fmt.Fprint(w, resStr)
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
