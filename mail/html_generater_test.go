package mail

import (
	"strings"
	"testing"

	"github.com/asendia/legacy-api/simple"
)

func TestGenerateReminderEmail(t *testing.T) {
	param := ReminderEmailParams{
		Title:              "Reminder to extend the delivery schedule of sejiwo.com testament",
		FullName:           "Asendia Mayco",
		InactiveAt:         simple.TimeTodayUTC().Add(simple.DaysToDuration(90)).Local().Format("YYYY-MM-DD"),
		TestamentReceivers: []string{"a@b.com", "c@d.com", "someone@somewhere.sometld"},
		ExtensionURL:       "https://sejiwo.com/extend?id=some-id&secret=some-secret",
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
		Title:        "A sejiwo.com message sent on behalf of Asendia Mayco",
		FullName:     "Asendia Mayco",
		EmailCreator: "noreply@sejiwo.com",
		MessageContentPerLine: strings.Split(`
`, "\n"),
		UnsubscribeURL: "https://sejiwo.com/unsubscribe?id=some-id&secret=some-secret",
	}
	content, err := GenerateTestamentEmail(param)
	if err != nil {
		t.Fatalf("Failed to generate testament email: %v", err)
	}
	if !strings.Contains(content, param.Title) {
		t.Fatal("Reminder email is not generated properly, missing title")
	}
}
