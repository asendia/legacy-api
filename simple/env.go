package simple

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func MustLoadEnv(filepath string) {
	if filepath == "" {
		env := DefaultString(os.Getenv("ENVIRONMENT"), "test")
		filepath = ".env-" + env + ".yaml"
	}
	err := godotenv.Load(filepath)
	if err != nil {
		log.Fatalf("Failed to load env: %v", err)
	}
}
