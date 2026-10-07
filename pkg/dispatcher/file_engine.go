package dispatcher

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/runtime"
)

type FileEngine struct{}

func NewFileEngine() *FileEngine {
	return &FileEngine{}
}

// ApplyFileOperations applique des transformations chirurgicales sans regex obscures
func (fe *FileEngine) ApplyFileOperations(filePath string, env *runtime.Environment) (*runtime.UnitInstance, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file '%s' does not exist", filePath)
	}

	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	content := string(contentBytes)

	// Création de l'unité FileContext pour exposer Replace, Json.Set, Json.Remove
	fileUnit := &runtime.UnitInstance{
		UnitName:   "FileContext",
		Properties: make(map[string]runtime.Value),
		Methods: map[string]runtime.NativeMethod{
			"Replace": func(args []runtime.Value) (runtime.Value, error) {
				if len(args) < 2 {
					return runtime.NilValue{}, fmt.Errorf("Replace requires target string and replacement")
				}
				target := args[0].String()
				repl := args[1].String()
				content = strings.ReplaceAll(content, target, repl)
				return runtime.BoolValue{Val: true}, os.WriteFile(filePath, []byte(content), 0644)
			},
			"JsonSet": func(args []runtime.Value) (runtime.Value, error) {
				if len(args) < 2 {
					return runtime.NilValue{}, fmt.Errorf("JsonSet requires key path and value")
				}
				keyPath := args[0].String()
				val := args[1].String()

				var data map[string]any
				if err := json.Unmarshal([]byte(content), &data); err != nil {
					return runtime.NilValue{}, fmt.Errorf("invalid json in %s: %w", filePath, err)
				}

				// Mise à jour de la clé (supporte "scripts.build")
				keys := strings.Split(keyPath, ".")
				setNestedJson(data, keys, val)

				updatedBytes, err := json.MarshalIndent(data, "", "  ")
				if err != nil {
					return runtime.NilValue{}, err
				}
				content = string(updatedBytes)
				return runtime.BoolValue{Val: true}, os.WriteFile(filePath, updatedBytes, 0644)
			},
			"JsonRemove": func(args []runtime.Value) (runtime.Value, error) {
				if len(args) < 1 {
					return runtime.NilValue{}, fmt.Errorf("JsonRemove requires key name")
				}
				key := args[0].String()

				var data map[string]any
				if err := json.Unmarshal([]byte(content), &data); err != nil {
					return runtime.NilValue{}, fmt.Errorf("invalid json in %s: %w", filePath, err)
				}

				delete(data, key)
				updatedBytes, err := json.MarshalIndent(data, "", "  ")
				if err != nil {
					return runtime.NilValue{}, err
				}
				content = string(updatedBytes)
				return runtime.BoolValue{Val: true}, os.WriteFile(filePath, updatedBytes, 0644)
			},
		},
		Directives: make(map[string]runtime.NativeDirective),
	}

	return fileUnit, nil
}

func setNestedJson(data map[string]any, keys []string, val any) {
	if len(keys) == 1 {
		data[keys[0]] = val
		return
	}
	currentKey := keys[0]
	subMap, exists := data[currentKey].(map[string]any)
	if !exists {
		subMap = make(map[string]any)
		data[currentKey] = subMap
	}
	setNestedJson(subMap, keys[1:], val)
}
