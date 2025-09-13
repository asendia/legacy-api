package p

import (
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
	corsStatus, err := api.VerifyCORS(w, r)
	if err != nil || corsStatus != http.StatusAccepted {
		return
	}
	// Verifying auth
	jwtRes, err := VerifyNetlifyJWT(r)
	if err != nil {
		log.Println(err.Error())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"err":"Decode and verify auth failed"}`, http.StatusForbidden)
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

	// Init API controller
	a := api.APIForFrontend{
		Context: ctx,
		Tx:      tx,
	}
	var res api.APIResponse
	action := r.URL.Query().Get("action")
	switch action {
	case "insert-message":
		p, errP := api.ParseReqInsertMessage(r)
		if errP != nil {
			err = errP
			break
		}
		res, err = a.InsertMessage(jwtRes, p)
	case "select-messages":
		res, err = a.SelectMessagesByEmailCreator(jwtRes)
	case "update-message":
		p, errP := api.ParseReqUpdateMessage(r)
		if errP != nil {
			err = errP
			break
		}
		res, err = a.UpdateMessage(jwtRes, p)
	case "delete-message":
		p, errP := api.ParseReqDeleteMessage(r)
		if errP != nil {
			err = errP
			break
		}
		res, err = a.DeleteMessage(jwtRes, p.ID)
	default:
		err = errors.New("invalid Action")
		res.StatusCode = http.StatusNotFound
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
		http.Error(w, `{"err":"Cannot generate a response"}`, http.StatusInternalServerError)
		return
	}
	tx.Commit(ctx)
	fmt.Fprint(w, resStr)
}

func VerifyNetlifyJWT(r *http.Request) (jwtRes secure.JWTResponse, err error) {
	// Always login with this email during test or cmd
	jwtRes = secure.JWTResponse{Email: "test@sejiwo.com"}
	// Verify JWT token on prod env only
	if os.Getenv("ENVIRONMENT") == "prod" {
		client := &http.Client{Timeout: time.Second * 10}
		jwtRes, err = secure.VerifyNetlifyJWT(client, r.Header.Get("authorization"))
		if err != nil {
			return
		}
	}
	return
}
