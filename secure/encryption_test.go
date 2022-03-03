package secure

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	secret, err := GenerateRandomString(32)
	if err != nil {
		t.Fatalf("Error generating random string: %v", err)
	}
	t.Logf("Random secret generated: %s", secret)
	plainText := "Hallo Inka, kamu adalah kentut"
	t.Logf("Encrypting text: %s\n", plainText)
	encryptResult, err := Encrypt(plainText, secret)
	if err != nil {
		t.Fatalf("Encryption error: %v\n", err)
	}
	t.Logf("Encryption success, now decrypting: %v\n", encryptResult)

	decryptedText, err := Decrypt(encryptResult, secret)
	if err != nil {
		t.Fatalf("Decryption error: %v\n", err)
	}
	t.Logf("Decryption success: %s\n", decryptedText)
	if plainText != decryptedText {
		t.Fatalf("Source string does not match with decrypted string:\n%s\n%s\n", plainText, decryptedText)
	}

	plainText = ""
	t.Logf("Encrypting text: %s\n", plainText)
	encryptResult, err = Encrypt(plainText, secret)
	if err != nil {
		t.Fatalf("Encryption error: %v\n", err)
	}
	t.Logf("Encryption success, now decrypting: %v\n", encryptResult)

	decryptedText, err = Decrypt(encryptResult, secret)
	if err != nil {
		t.Fatalf("Decryption error: %v\n", err)
	}
	t.Logf("Decryption success: %s\n", decryptedText)
	if plainText != decryptedText {
		t.Fatalf("Source string does not match with decrypted string:\n%s\n%s\n", plainText, decryptedText)
	}
}
