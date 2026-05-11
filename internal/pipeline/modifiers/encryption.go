package modifiers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"time"

	"golang.org/x/crypto/chacha20poly1305"

	"socks5-dpi-proxy/internal/pipeline"
)

type EncryptionModifier struct {
	Type             string
	Key              string
	aead             cipher.AEAD
	lastRotation     time.Time
	rotationInterval time.Duration
}

func (e *EncryptionModifier) Name() string {
	return "encryption"
}

func (e *EncryptionModifier) Configure(config map[string]interface{}) error {
	if encType, ok := config["type"].(string); ok {
		e.Type = encType
	} else {
		e.Type = "xor"
	}

	if key, ok := config["key"].(string); ok {
		e.Key = key
	} else {
		e.Key = "default_key"
	}

	// Initialize cipher based on type
	switch e.Type {
	case "chacha20poly1305":
		key := e.deriveKey([]byte(e.Key), 32) // ChaCha20-Poly1305 needs 32-byte key
		aead, err := chacha20poly1305.New(key)
		if err != nil {
			return fmt.Errorf("failed to initialize ChaCha20-Poly1305: %w", err)
		}
		e.aead = aead
	case "aes-gcm":
		key := e.deriveKey([]byte(e.Key), 32) // AES-256 needs 32-byte key
		block, err := aes.NewCipher(key)
		if err != nil {
			return fmt.Errorf("failed to initialize AES cipher: %w", err)
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return fmt.Errorf("failed to initialize AES-GCM: %w", err)
		}
		e.aead = aead
	case "xor":
		// XOR doesn't need initialization (kept for backward compatibility)
	default:
		return fmt.Errorf("unsupported encryption type: %s", e.Type)
	}

	// Set default rotation interval (1 hour)
	if e.rotationInterval == 0 {
		e.rotationInterval = time.Hour
	}
	e.lastRotation = time.Now()

	return nil
}

func (e *EncryptionModifier) Process(data []byte, direction pipeline.Direction) []byte {
	if len(data) == 0 {
		return data
	}

	// Check if key rotation is needed
	if e.shouldRotateKey() {
		e.rotateKey()
	}

	switch e.Type {
	case "chacha20poly1305":
		return e.chacha20Process(data, direction)
	case "aes-gcm":
		return e.aesGcmProcess(data, direction)
	case "xor":
		return e.xorProcess(data, direction)
	default:
		return data
	}
}

func (e *EncryptionModifier) xorProcess(data []byte, direction pipeline.Direction) []byte {
	keyBytes := []byte(e.Key)
	result := make([]byte, len(data))

	for i, b := range data {
		keyByte := keyBytes[i%len(keyBytes)]
		if direction == pipeline.DirectionOutbound {
			result[i] = b ^ keyByte
		} else {
			result[i] = b ^ keyByte // XOR is symmetric
		}
	}

	return result
}

func (e *EncryptionModifier) chacha20Process(data []byte, direction pipeline.Direction) []byte {
	if e.aead == nil {
		return data
	}

	if direction == pipeline.DirectionOutbound {
		// Encrypt: generate nonce and prepend to ciphertext
		nonce := make([]byte, e.aead.NonceSize())
		if _, err := rand.Read(nonce); err != nil {
			return data
		}

		result := e.aead.Seal(nonce, nonce, data, nil)
		return result
	} else {
		// Decrypt: extract nonce from beginning of data
		nonceSize := e.aead.NonceSize()
		if len(data) < nonceSize {
			return data
		}

		nonce := data[:nonceSize]
		ciphertext := data[nonceSize:]

		result, err := e.aead.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return data // Return original data if decryption fails
		}
		return result
	}
}

func (e *EncryptionModifier) aesGcmProcess(data []byte, direction pipeline.Direction) []byte {
	if e.aead == nil {
		return data
	}

	if direction == pipeline.DirectionOutbound {
		// Encrypt: generate nonce and prepend to ciphertext
		nonce := make([]byte, e.aead.NonceSize())
		if _, err := rand.Read(nonce); err != nil {
			return data
		}

		result := e.aead.Seal(nonce, nonce, data, nil)
		return result
	} else {
		// Decrypt: extract nonce from beginning of data
		nonceSize := e.aead.NonceSize()
		if len(data) < nonceSize {
			return data
		}

		nonce := data[:nonceSize]
		ciphertext := data[nonceSize:]

		result, err := e.aead.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return data // Return original data if decryption fails
		}
		return result
	}
}

// deriveKey creates a proper-length key from the input key
func (e *EncryptionModifier) deriveKey(inputKey []byte, length int) []byte {
	key := make([]byte, length)
	copy(key, inputKey)

	// If input key is shorter than needed, repeat it
	for i := length; i < len(inputKey); i++ {
		key[i%length] = inputKey[i%len(inputKey)] ^ byte(i)
	}

	return key
}

// shouldRotateKey checks if key rotation is needed
func (e *EncryptionModifier) shouldRotateKey() bool {
	return e.Type != "xor" && time.Since(e.lastRotation) > e.rotationInterval
}

// rotateKey performs key rotation
func (e *EncryptionModifier) rotateKey() {
	// Generate new key based on current key and timestamp
	newKeyData := []byte(e.Key + time.Now().Format("20060102150405"))
	newKey := e.deriveKey(newKeyData, 32)

	switch e.Type {
	case "chacha20poly1305":
		aead, err := chacha20poly1305.New(newKey)
		if err == nil {
			e.aead = aead
		}
	case "aes-gcm":
		block, err := aes.NewCipher(newKey)
		if err == nil {
			aead, err := cipher.NewGCM(block)
			if err == nil {
				e.aead = aead
			}
		}
	}

	e.lastRotation = time.Now()
}
