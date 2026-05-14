package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	// Тестовые случаи
	testCases := []struct {
		name    string
		message string
	}{
		{"Empty message", ""},
		{"Short message", "Hello, World!"},
		{"Long message", "This is a longer message that spans multiple sentences. It contains various characters and should be encrypted and decrypted correctly. The purpose is to test the encryption and decryption functions with a larger input."},
		{"Special characters", "!@#$%^&*()_+{}|:<>?~`-=[]\\;',./"},
		{"Non-English characters", "Привет, мир! こんにちは世界! 你好，世界！"},
	}

	// Генерируем ключ
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Шифруем сообщение
			encrypted, err := EncryptMessage(tc.message, key)
			if err != nil {
				t.Fatalf("Failed to encrypt message: %v", err)
			}

			// Проверяем, что зашифрованное сообщение отличается от исходного
			if tc.message != "" && bytes.Equal([]byte(tc.message), encrypted) {
				t.Errorf("Encrypted message is identical to original message")
			}

			// Дешифруем сообщение
			decrypted, err := DecryptMessage(encrypted, key)
			if err != nil {
				t.Fatalf("Failed to decrypt message: %v", err)
			}

			// Проверяем, что дешифрованное сообщение совпадает с исходным
			if decrypted != tc.message {
				t.Errorf("Decrypted message does not match original message. Got: %s, Want: %s", decrypted, tc.message)
			}
		})
	}
}

func TestEncryptWithWrongKey(t *testing.T) {
	// Генерируем два разных ключа
	key1, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key1: %v", err)
	}

	key2, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key2: %v", err)
	}

	// Проверяем, что ключи разные
	if bytes.Equal(key1, key2) {
		t.Fatalf("Generated keys are identical, which is extremely unlikely")
	}

	// Тестовое сообщение
	message := "This is a secret message"

	// Шифруем сообщение с первым ключом
	encrypted, err := EncryptMessage(message, key1)
	if err != nil {
		t.Fatalf("Failed to encrypt message: %v", err)
	}

	// Пытаемся дешифровать с другим ключом (должно вернуть ошибку аутентификации)
	_, err = DecryptMessage(encrypted, key2)
	if err == nil {
		t.Errorf("Expected error when decrypting with wrong key (GCM authentication should fail), but got none")
	}
}

func TestGenerateKey(t *testing.T) {
	// Генерируем несколько ключей и проверяем их свойства
	numKeys := 5
	keys := make([][]byte, numKeys)

	for i := 0; i < numKeys; i++ {
		key, err := GenerateKey()
		if err != nil {
			t.Fatalf("Failed to generate key %d: %v", i, err)
		}

		// Проверяем длину ключа (32 байта для AES-256)
		if len(key) != 32 {
			t.Errorf("Key %d has incorrect length: got %d, want 32", i, len(key))
		}

		keys[i] = key
	}

	// Проверяем, что все ключи уникальны
	for i := 0; i < numKeys; i++ {
		for j := i + 1; j < numKeys; j++ {
			if bytes.Equal(keys[i], keys[j]) {
				t.Errorf("Keys %d and %d are identical, which is extremely unlikely", i, j)
			}
		}
	}
}

func TestDecryptInvalidData(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	testCases := []struct {
		name        string
		ciphertext  []byte
		expectError bool
	}{
		{"Empty data", []byte{}, true},
		{"Too short data", []byte{1, 2, 3}, true},
		{"Random data", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecryptMessage(tc.ciphertext, key)

			if tc.expectError && err == nil {
				t.Errorf("Expected error when decrypting invalid data, but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Did not expect error when decrypting data, but got: %v", err)
			}
		})
	}
}
