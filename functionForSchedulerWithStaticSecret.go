package p

import (
	"encoding/json"
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
	apiReq, statusCode, err := DecodeAndVerifySchedulerSecret(r)
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
	switch apiReq.Action {
	case "select-messages-need-reminding":
		res, err = a.SelectMessagesNeedReminding()
		break
	case "update-message-after-sending-reminder":
		res, err = a.UpdateMessageAfterSendingReminder(apiReq.Data.ID)
		break
	case "select-inactive-messages":
		res, err = a.SelectInactiveMessages()
		break
	case "update-message-after-sending-testament":
		res, err = a.UpdateMessageAfterSendingTestament(apiReq.Data.ID)
		break
	default:
		err = errors.New("Invalid Action")
		res.StatusCode = http.StatusNotFound
		break
	}
	// Handle controller error
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), res.StatusCode)
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

func DecodeAndVerifySchedulerSecret(r *http.Request) (req api.APIRequestMessageData, statusCode int, err error) {
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		statusCode = http.StatusBadRequest
		return
	}
	serverSecret := os.Getenv("STATIC_SECRET")
	if len(serverSecret) != api.ExtensionSecretLength {
		statusCode = http.StatusInternalServerError
		err = errors.New("env STATIC_SECRET is invalid: " + serverSecret)
		return
	}
	clientSecret := r.Header.Get("x-static-secret")
	if serverSecret != clientSecret {
		statusCode = http.StatusUnauthorized
		err = errors.New("Invalid x-static-secret header: " + clientSecret)
		return
	}
	return req, http.StatusOK, nil
}
