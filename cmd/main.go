package main

import (
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
	http.HandleFunc("/legacy-api-scheduler", p.CloudFunctionForSchedulerWithStaticSecret)
	log.Printf("Server is running on localhost:%d", port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), nil))
}
