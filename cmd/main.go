package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	p "github.com/asendia/legacy-api"
	"github.com/asendia/legacy-api/simple"
)

func main() {
	simple.MustLoadEnv("")
	port := 8080
	http.HandleFunc("/legacy-api", p.CloudFunctionForFrontendWithNetlifyJWT)
	http.HandleFunc("/legacy-api-secret", p.CloudFunctionForFrontendWithUserSecret)
	http.HandleFunc("/legacy-api-scheduler", handleScheduler)
	log.Printf("Server is running on localhost:%d", port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), nil))
}

func handleScheduler(w http.ResponseWriter, r *http.Request) {
	var psm p.PubSubMessage
	json.NewDecoder(r.Body).Decode(&psm)
	err := p.CloudFunctionForSchedulerWithStaticSecret(r.Context(), psm)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %+v\n", err), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Success!")
}
