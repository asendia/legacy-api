package mail

import (
	"os"
	"testing"

	"github.com/asendia/legacy-api/simple"
)

func TestMain(m *testing.M) {
	simple.MustLoadEnv("../.env-test.yaml")
	code := m.Run()
	os.Exit(code)
}
