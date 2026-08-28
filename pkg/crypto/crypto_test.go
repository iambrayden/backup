package crypto_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/zeebo/blake3"

	"github.com/waylontech/backup/pkg/crypto"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	dek, err := crypto.GenerateDEK()
	if err != nil {
		t.Fatal(err)
	}

	plaintext := make([]byte, 1<<20)
	rand.Read(plaintext)

	hash := blake3.Sum256(plaintext)

	encrypted, err := crypto.EncryptChunk(dek, hash, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := crypto.DecryptChunk(dek, hash, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("decrypted data does not match original")
	}
}

func TestEncryptProducesNonDeterministicCiphertext(t *testing.T) {
	dek, _ := crypto.GenerateDEK()
	plaintext := make([]byte, 4096)
	rand.Read(plaintext)

	hash := blake3.Sum256(plaintext)

	ct1, _ := crypto.EncryptChunk(dek, hash, plaintext)
	ct2, _ := crypto.EncryptChunk(dek, hash, plaintext)

	if bytes.Equal(ct1, ct2) {
		t.Error("expected different ciphertexts due to random nonces")
	}
}

func TestDifferentDEKsProduceDifferentCiphertexts(t *testing.T) {
	dek1, _ := crypto.GenerateDEK()
	dek2, _ := crypto.GenerateDEK()

	plaintext := make([]byte, 4096)
	rand.Read(plaintext)

	hash := blake3.Sum256(plaintext)

	ct1, _ := crypto.EncryptChunk(dek1, hash, plaintext)
	ct2, _ := crypto.EncryptChunk(dek2, hash, plaintext)

	if bytes.Equal(ct1, ct2) {
		t.Error("expected different ciphertexts for different DEKs")
	}
}

func TestWrongDEKFailsDecrypt(t *testing.T) {
	dek1, _ := crypto.GenerateDEK()
	dek2, _ := crypto.GenerateDEK()

	plaintext := make([]byte, 4096)
	rand.Read(plaintext)

	hash := blake3.Sum256(plaintext)

	encrypted, _ := crypto.EncryptChunk(dek1, hash, plaintext)

	_, err := crypto.DecryptChunk(dek2, hash, encrypted)
	if err == nil {
		t.Error("expected error decrypting with wrong DEK")
	}
}

func TestDEKEnvelopeEncryption(t *testing.T) {
	masterKey := make([]byte, 32)
	rand.Read(masterKey)

	dek, err := crypto.GenerateDEK()
	if err != nil {
		t.Fatal(err)
	}

	encryptedDEK, err := crypto.EncryptDEK(masterKey, dek)
	if err != nil {
		t.Fatalf("encrypt dek: %v", err)
	}

	decryptedDEK, err := crypto.DecryptDEK(masterKey, encryptedDEK)
	if err != nil {
		t.Fatalf("decrypt dek: %v", err)
	}

	if !bytes.Equal(dek, decryptedDEK) {
		t.Fatal("decrypted DEK does not match original")
	}
}

func TestWrongMasterKeyFailsDEKDecrypt(t *testing.T) {
	masterKey := make([]byte, 32)
	rand.Read(masterKey)
	wrongKey := make([]byte, 32)
	rand.Read(wrongKey)

	dek, _ := crypto.GenerateDEK()
	blob, _ := crypto.EncryptDEK(masterKey, dek)

	_, err := crypto.DecryptDEK(wrongKey, blob)
	if err == nil {
		t.Error("expected error with wrong master key")
	}
}

func BenchmarkEncryptChunk(b *testing.B) {
	dek, _ := crypto.GenerateDEK()
	data := make([]byte, 1<<20)
	rand.Read(data)
	hash := blake3.Sum256(data)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		crypto.EncryptChunk(dek, hash, data)
	}
}
