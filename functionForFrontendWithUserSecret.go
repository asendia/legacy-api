package p

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/asendia/legacy-api/api"
	"github.com/asendia/legacy-api/data"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// Google Cloud Function
func CloudFunctionForFrontendWithUserSecret(w http.ResponseWriter, r *http.Request) {
	corsStatus, err := api.VerifyCORS(w, r)
	if err != nil || corsStatus != http.StatusAccepted {
		return
	}
	godotenv.Load()
	secret, messageID, err := VerifyQueryString(r)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Invalid query string", http.StatusForbidden)
		return
	}

	// Establishing connection to database
	ctx := r.Context()
	conn, err := data.ConnectDB(ctx, data.LoadDBURLConfig())
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
	action := r.URL.Query().Get("action")
	switch action {
	case "extend-message":
		res, err = a.ExtendMessageInactiveAt(secret, messageID)
		break
	case "unsubscribe-message":
		res, err = a.UnsubscribeMessage(secret, messageID)
		break
	default:
		err = errors.New("Invalid Action")
		res.StatusCode = http.StatusNotFound
		res.ResponseMsg = err.Error()
		break
	}
	w.Header().Set("Content-Type", "application/json")
	// Handle controller error
	if err != nil {
		log.Println(err.Error())
		http.Error(w, `{"err":"`+err.Error()+`"}`, res.GetValidStatusCode())
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

func VerifyQueryString(r *http.Request) (secret string, id uuid.UUID, err error) {
	q := r.URL.Query()
	id, err = uuid.Parse(q.Get("id"))
	if err != nil {
		return secret, id, err
	}
	secret = q.Get("secret")
	if len(secret) != api.ExtensionSecretLength {
		return secret, id, errors.New("Invalid secret")
	}
	return secret, id, err
}
