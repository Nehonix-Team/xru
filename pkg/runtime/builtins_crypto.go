package runtime

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

func generateSecureRandomBytes(length int) (string, error) {
	if length <= 0 {
		length = 32
	}
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate secure random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func newCryptoUnit() *UnitInstance {
	return &UnitInstance{
		UnitName:   "Crypto",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"GetRandomBytes": func(args []Value) (Value, error) {
				length := 32
				if len(args) > 0 {
					if n, ok := args[0].(NumberValue); ok {
						length = int(n.Val)
					}
				}
				hexStr, err := generateSecureRandomBytes(length)
				if err != nil {
					return NilValue{}, err
				}
				return StringValue{Val: hexStr}, nil
			},
			"RandomBytes": func(args []Value) (Value, error) {
				length := 32
				if len(args) > 0 {
					if n, ok := args[0].(NumberValue); ok {
						length = int(n.Val)
					}
				}
				hexStr, err := generateSecureRandomBytes(length)
				if err != nil {
					return NilValue{}, err
				}
				return StringValue{Val: hexStr}, nil
			},
			"RandomHex": func(args []Value) (Value, error) {
				length := 32
				if len(args) > 0 {
					if n, ok := args[0].(NumberValue); ok {
						length = int(n.Val)
					}
				}
				hexStr, err := generateSecureRandomBytes(length)
				if err != nil {
					return NilValue{}, err
				}
				return StringValue{Val: hexStr}, nil
			},
			"RandomBase64": func(args []Value) (Value, error) {
				length := 32
				if len(args) > 0 {
					if n, ok := args[0].(NumberValue); ok {
						length = int(n.Val)
					}
				}
				if length <= 0 {
					length = 32
				}
				b := make([]byte, length)
				if _, err := rand.Read(b); err != nil {
					return NilValue{}, err
				}
				return StringValue{Val: base64.StdEncoding.EncodeToString(b)}, nil
			},
			"Sha256": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Crypto.Sha256 requires input string")
				}
				h := sha256.Sum256([]byte(args[0].String()))
				return StringValue{Val: hex.EncodeToString(h[:])}, nil
			},
			"Uuid": func(args []Value) (Value, error) {
				b := make([]byte, 16)
				if _, err := rand.Read(b); err != nil {
					return NilValue{}, err
				}
				b[6] = (b[6] & 0x0f) | 0x40
				b[8] = (b[8] & 0x3f) | 0x80
				uuid := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
					b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
				return StringValue{Val: uuid}, nil
			},
		},
		Directives: make(map[string]NativeDirective),
	}
}
