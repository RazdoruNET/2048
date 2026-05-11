package pipeline

import (
	"crypto/rc4"
	"fmt"
)

type EncryptionModifier struct {
	Type string
	Key  string
	cipher *rc4.Cipher
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
	case "rc4":
		cipher, err := rc4.NewCipher([]byte(e.Key))
		if err != nil {
			return fmt.Errorf("failed to initialize RC4 cipher: %w", err)
		}
		e.cipher = cipher
	case "xor":
		// XOR doesn't need initialization
	default:
		return fmt.Errorf("unsupported encryption type: %s", e.Type)
	}

	return nil
}

func (e *EncryptionModifier) Process(data []byte, direction Direction) []byte {
	if len(data) == 0 {
		return data
	}

	switch e.Type {
	case "xor":
		return e.xorProcess(data, direction)
	case "rc4":
		return e.rc4Process(data, direction)
	default:
		return data
	}
}

func (e *EncryptionModifier) xorProcess(data []byte, direction Direction) []byte {
	keyBytes := []byte(e.Key)
	result := make([]byte, len(data))
	
	for i, b := range data {
		keyByte := keyBytes[i%len(keyBytes)]
		if direction == DirectionOutbound {
			result[i] = b ^ keyByte
		} else {
			result[i] = b ^ keyByte // XOR is symmetric
		}
	}
	
	return result
}

func (e *EncryptionModifier) rc4Process(data []byte, direction Direction) []byte {
	if e.cipher == nil {
		return data
	}

	// For RC4, we need to create a new cipher for each message to maintain state
	cipher, err := rc4.NewCipher([]byte(e.Key))
	if err != nil {
		return data
	}

	result := make([]byte, len(data))
	cipher.XORKeyStream(result, data)
	
	return result
}
