package mail

import (
	"fmt"
	"strings"
	"testing"

	"github.com/asendia/legacy-api/simple"
)

func TestGenerateReminderEmail(t *testing.T) {
	param := ReminderEmailParams{
		Title:          "Reminder to extend the delivery schedule of warisin.com testament",
		FullName:       "Asendia Mayco",
		InactiveAt:     simple.TimeTodayUTC().Add(simple.DaysToDuration(90)).Local().Format("YYYY-MM-DD"),
		EmailReceivers: []string{"a@b.com", "c@d.com", "someone@somewhere.sometld"},
		ExtensionURL:   "https://warisin.com/extend?id=some-id&secret=some-secret",
	}
	content, err := GenerateReminderEmail(param)
	if err != nil {
		t.Fatalf("Failed to generate reminder email: %v", err)
	}
	if !strings.Contains(content, param.Title) {
		t.Fatal("Reminder email is not generated properly, missing title")
	}
}

func TestGenerateTestamentEmail(t *testing.T) {
	param := TestamentEmailParams{
		Title:        fmt.Sprintf("A warisin.com message sent on behalf of Asendia Mayco"),
		FullName:     "Asendia Mayco",
		EmailCreator: "noreply@warisin.com",
		MessageContentPerLine: strings.Split(`
`, "\n"),
		UnsubscribeURL: "https://warisin.com/unsubscribe?id=some-id&secret=some-secret",
	}
	content, err := GenerateTestamentEmail(param)
	if err != nil {
		t.Fatalf("Failed to generate testament email: %v", err)
	}
	if !strings.Contains(content, param.Title) {
		t.Fatal("Reminder email is not generated properly, missing title")
	}
}
