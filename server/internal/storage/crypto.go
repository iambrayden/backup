package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

func encryptBytes(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ct := gcm.Seal(nil, nonce, plaintext, nil)
	out := make([]byte, len(nonce)+len(ct))
	copy(out, nonce)
	copy(out[len(nonce):], ct)
	return out, nil
}

func decryptBytes(key, blob []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(blob) < gcm.NonceSize() {
		return nil, errors.New("blob too short")
	}
	return gcm.Open(nil, blob[:gcm.NonceSize()], blob[gcm.NonceSize():], nil)
}
