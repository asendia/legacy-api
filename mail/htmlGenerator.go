package mail

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
)

type ReminderEmailParams struct {
	Title              string
	FullName           string
	InactiveAt         string
	TestamentReceivers []string
	ExtensionURL       string
}

func GenerateReminderEmail(param ReminderEmailParams) (string, error) {
	t, err := template.New("template-reminder.html").ParseFiles(
		generateTemplateDir("template-reminder.html"))
	if err != nil {
		return "", err
	}
	var result bytes.Buffer
	err = t.ExecuteTemplate(&result, "template-reminder.html", param)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

type TestamentEmailParams struct {
	Title                 string
	FullName              string
	EmailCreator          string
	MessageContentPerLine []string
	UnsubscribeURL        string
	HowToDecrypt          string
}

func GenerateTestamentEmail(param TestamentEmailParams) (string, error) {
	t, err := template.New("template-testament.html").ParseFiles(
		generateTemplateDir("template-testament.html"))
	if err != nil {
		return "", err
	}
	var result bytes.Buffer
	err = t.ExecuteTemplate(&result, "template-testament.html", param)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

func generateTemplateDir(filename string) string {
	// Google cloud function
	// This took me 1 hour to debug https://cloud.google.com/functions/docs/concepts/exec#file_system
	var rootDir = os.Getenv("SERVERLESS_FUNCTION_SOURCE_CODE")
	mailDir := "mail/"
	if os.Getenv("ENVIRONMENT") == "test" {
		mailDir = ""
	}
	fmt.Println(mailDir)
	return rootDir + mailDir + filename
}
