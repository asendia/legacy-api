package api

import "testing"

func TestEncryptDecryptClientEncryptedMessage(t *testing.T) {
	message := "aes.utf8:thisisclientencrypteddummy"
	key := "uselesssecret"
	result, err := EncryptMessageContent(message, key)
	if err != nil {
		t.Fatalf("EncryptMessageContent is failed to handle client encrypted message")
	}
	if message != result {
		t.Fatal("Client encrypted message should not be encrypted again")
	}
	result, err = DecryptMessageContent(result, key)
	if err != nil {
		t.Fatalf("DecryptMessageContent is failed to handle client encrypted message")
	}
	if message != result {
		t.Fatal("Client encrypted message should be passed as is")
	}
}
