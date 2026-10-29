package runtime

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
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

// hashFile computes the cryptographic hash of a file at filePath in a streaming, cross-platform manner.
func hashFile(filePath string, algo string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file '%s': %w", filePath, err)
	}
	defer f.Close()

	var h hash.Hash
	switch strings.ToLower(algo) {
	case "sha256", "":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	case "md5":
		h = md5.New()
	case "sha1":
		h = sha1.New()
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", algo)
	}

	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to compute hash for '%s': %w", filePath, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func newCryptoUnit() *UnitInstance {
	getRandomBytesFn := func(args []Value) (Value, error) {
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
	}

	randomBase64Fn := func(args []Value) (Value, error) {
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
	}

	sha256Fn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Crypto.Sha256 requires input string")
		}
		h := sha256.Sum256([]byte(args[0].String()))
		return StringValue{Val: hex.EncodeToString(h[:])}, nil
	}

	sha512Fn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Crypto.Sha512 requires input string")
		}
		h := sha512.Sum512([]byte(args[0].String()))
		return StringValue{Val: hex.EncodeToString(h[:])}, nil
	}

	md5Fn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Crypto.Md5 requires input string")
		}
		h := md5.Sum([]byte(args[0].String()))
		return StringValue{Val: hex.EncodeToString(h[:])}, nil
	}

	fileSha256Fn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Crypto.fileSha256 requires filePath")
		}
		filePath := args[0].String()
		hashStr, err := hashFile(filePath, "sha256")
		if err != nil {
			return NilValue{}, err
		}
		return StringValue{Val: hashStr}, nil
	}

	fileHashFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Crypto.fileHash requires filePath")
		}
		filePath := args[0].String()
		algo := "sha256"
		if len(args) >= 2 {
			algo = strings.TrimSpace(args[1].String())
		}
		hashStr, err := hashFile(filePath, algo)
		if err != nil {
			return NilValue{}, err
		}
		return StringValue{Val: hashStr}, nil
	}

	verifyFileFn := func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("Crypto.verifyFile requires filePath and expectedHash")
		}
		filePath := args[0].String()
		expected := strings.TrimSpace(strings.ToLower(args[1].String()))
		algo := "sha256"
		if len(args) >= 3 {
			algo = strings.TrimSpace(args[2].String())
		}
		actual, err := hashFile(filePath, algo)
		if err != nil {
			return BoolValue{Val: false}, err
		}
		match := subtle.ConstantTimeCompare([]byte(strings.ToLower(actual)), []byte(expected)) == 1
		return BoolValue{Val: match}, nil
	}

	verifyFn := func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("Crypto.verify requires actual and expected values")
		}
		actual := strings.TrimSpace(strings.ToLower(args[0].String()))
		expected := strings.TrimSpace(strings.ToLower(args[1].String()))
		match := subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
		return BoolValue{Val: match}, nil
	}

	uuidFn := func(args []Value) (Value, error) {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return NilValue{}, err
		}
		b[6] = (b[6] & 0x0f) | 0x40
		b[8] = (b[8] & 0x3f) | 0x80
		uuid := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
		return StringValue{Val: uuid}, nil
	}

	return &UnitInstance{
		UnitName:   "Crypto",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"GetRandomBytes": getRandomBytesFn,
			"getRandomBytes": getRandomBytesFn,
			"RandomBytes":    getRandomBytesFn,
			"randomBytes":    getRandomBytesFn,
			"RandomHex":      getRandomBytesFn,
			"randomHex":      getRandomBytesFn,
			"RandomBase64":   randomBase64Fn,
			"randomBase64":   randomBase64Fn,
			"Sha256":         sha256Fn,
			"sha256":         sha256Fn,
			"Sha512":         sha512Fn,
			"sha512":         sha512Fn,
			"Md5":            md5Fn,
			"md5":            md5Fn,
			"FileSha256":     fileSha256Fn,
			"fileSha256":     fileSha256Fn,
			"Sha256File":     fileSha256Fn,
			"sha256File":     fileSha256Fn,
			"FileHash":       fileHashFn,
			"fileHash":       fileHashFn,
			"VerifyFile":     verifyFileFn,
			"verifyFile":     verifyFileFn,
			"VerifyFileHash": verifyFileFn,
			"verifyFileHash": verifyFileFn,
			"Verify":         verifyFn,
			"verify":         verifyFn,
			"VerifyHash":     verifyFn,
			"verifyHash":     verifyFn,
			"Uuid":           uuidFn,
			"uuid":           uuidFn,
		},
		Directives: make(map[string]NativeDirective),
	}
}

