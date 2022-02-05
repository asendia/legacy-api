package p

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/asendia/legacy-api/api"
	"github.com/asendia/legacy-api/data"
	"github.com/asendia/legacy-api/secure"
)

// Google Cloud Function
func CloudFunctionForFrontendWithNetlifyJWT(w http.ResponseWriter, r *http.Request) {
	// Verifying auth
	apiReq, statusCode, err := DecodeAndVerifyNetlifyJWT(r)
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

	// Init API controller
	a := api.APIForFrontend{
		Context: ctx,
		Tx:      tx,
	}
	var res api.APIResponse
	switch apiReq.Action {
	case "insert-message":
		res, err = a.InsertMessage(apiReq.Data)
		break
	case "select-message-by-id":
		res, err = a.SelectMessageByID(apiReq.Data.ID)
		break
	case "select-messages-by-email-creator":
		res, err = a.SelectMessagesByEmailCreator(apiReq.Data.EmailCreator)
		break
	case "update-message":
		res, err = a.UpdateMessage(data.UpdateMessageParams{
			EmailCreator:         apiReq.Data.EmailCreator,
			EmailReceivers:       apiReq.Data.EmailReceivers,
			MessageContent:       apiReq.Data.MessageContent,
			InactivePeriodDays:   apiReq.Data.InactivePeriodDays,
			ReminderIntervalDays: apiReq.Data.ReminderIntervalDays,
			IsActive:             apiReq.Data.IsActive,
			ID:                   apiReq.Data.ID,
		})
		break
	case "delete-message":
		res, err = a.DeleteMessage(apiReq.Data.ID)
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

func DecodeAndVerifyNetlifyJWT(r *http.Request) (req api.APIRequestMessageData, statusCode int, err error) {
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		statusCode = http.StatusBadRequest
		return
	}
	// Always login with this email during test or cmd
	jwtRes := &secure.JWTResponse{Email: "mock@mock"}
	// Verify JWT token on prod env only
	if os.Getenv("ENVIRONMENT") == "prod" {
		client := &http.Client{Timeout: time.Second * 10}
		jwtRes, err = secure.VerifyNetlifyJWT(client, r.Header.Get("authorization"))
		if err != nil {
			statusCode = http.StatusUnauthorized
			return
		}
	}
	// Verify that jwtRes.Email matches the d.Data.EmailCreator
	if jwtRes.Email != req.Data.EmailCreator {
		statusCode = http.StatusForbidden
		err = errors.New(fmt.Sprintf("Unauthorized mismatch email, jwtRes.Email: %s, d.Data.EmailCreator: %s\n",
			jwtRes.Email, req.Data.EmailCreator))
		return
	}
	return
}
