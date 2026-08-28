package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
)

var (
	zstdEncoder *zstd.Encoder
	zstdDecoder *zstd.Decoder
)

func init() {
	var err error
	zstdEncoder, err = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		panic(err)
	}
	zstdDecoder, err = zstd.NewReader(nil)
	if err != nil {
		panic(err)
	}
}

func DeriveChunkKey(dek []byte, chunkHash [32]byte) ([]byte, error) {
	if len(dek) != 32 {
		return nil, fmt.Errorf("dek must be 32 bytes, got %d", len(dek))
	}
	h := sha256.New()
	h.Write(dek)
	h.Write(chunkHash[:])
	h.Write([]byte("chunk-key-v1"))
	return h.Sum(nil), nil
}

func EncryptChunk(dek []byte, chunkHash [32]byte, plaintext []byte) ([]byte, error) {
	compressed := zstdEncoder.EncodeAll(plaintext, nil)

	key, err := DeriveChunkKey(dek, chunkHash)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, compressed, nil)
	out := make([]byte, len(nonce)+len(ciphertext))
	copy(out, nonce)
	copy(out[len(nonce):], ciphertext)
	return out, nil
}

func DecryptChunk(dek []byte, chunkHash [32]byte, blob []byte) ([]byte, error) {
	key, err := DeriveChunkKey(dek, chunkHash)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}

	if len(blob) < gcm.NonceSize() {
		return nil, errors.New("blob too short")
	}

	nonce := blob[:gcm.NonceSize()]
	ciphertext := blob[gcm.NonceSize():]

	compressed, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	plain, err := zstdDecoder.DecodeAll(compressed, nil)
	if err != nil {
		return nil, fmt.Errorf("decompress: %w", err)
	}
	return plain, nil
}

func GenerateDEK() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func EncryptDEK(masterKey, dek []byte) ([]byte, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("master key must be 32 bytes")
	}
	block, err := aes.NewCipher(masterKey)
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
	ct := gcm.Seal(nil, nonce, dek, nil)
	out := make([]byte, len(nonce)+len(ct))
	copy(out, nonce)
	copy(out[len(nonce):], ct)
	return out, nil
}

func DecryptDEK(masterKey, blob []byte) ([]byte, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("master key must be 32 bytes")
	}
	block, err := aes.NewCipher(masterKey)
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
	nonce := blob[:gcm.NonceSize()]
	ct := blob[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ct, nil)
}
