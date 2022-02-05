package p

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/asendia/legacy-api/api"
	"github.com/asendia/legacy-api/data"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// Google Cloud Function
func CloudFunctionForFrontendWithUserSecret(w http.ResponseWriter, r *http.Request) {
	godotenv.Load()
	apiReq, statusCode, err := DecodeAndVerifyUserSecret(r)
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

	a := api.APIForFrontend{
		Context: ctx,
		Tx:      tx,
	}
	var res api.APIResponse
	switch apiReq.Action {
	case "extend-message":
		res, err = a.ExtendMessageInactiveAt(apiReq.Data.ID, apiReq.Data.ExtensionSecret)
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

func DecodeAndVerifyUserSecret(r *http.Request) (req api.APIRequestMessageData, statusCode int, err error) {
	q := r.URL.Query()
	id, err := uuid.Parse(q.Get("id"))
	if err != nil {
		return req, http.StatusBadRequest, err
	}
	secret := q.Get("secret")
	if len(secret) != api.ExtensionSecretLength {
		return req, http.StatusBadRequest, errors.New("Invalid secret")
	}
	return api.APIRequestMessageData{
		Action: q.Get("action"),
		Data: api.MessageData{
			ID:              id,
			ExtensionSecret: secret,
		},
	}, http.StatusOK, nil
}
