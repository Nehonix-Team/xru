package runtime

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func newFsUnit() *UnitInstance {
	ensureDirFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Fs.ensureDir requires directory path")
		}
		target := args[0].String()
		if err := os.MkdirAll(target, 0755); err != nil {
			return NilValue{}, fmt.Errorf("Fs.ensureDir failed: %w", err)
		}
		return BoolValue{Val: true}, nil
	}

	removeFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Fs.remove requires target path")
		}
		target := args[0].String()
		if _, err := os.Lstat(target); err != nil {
			return NilValue{}, fmt.Errorf("Fs.remove failed: %w", err)
		}
		if err := os.RemoveAll(target); err != nil {
			return NilValue{}, fmt.Errorf("Fs.remove failed: %w", err)
		}
		return BoolValue{Val: true}, nil
	}

	moveFn := func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("Fs.move requires src and dest path")
		}
		src, dest := args[0].String(), args[1].String()
		if err := os.Rename(src, dest); err != nil {
			return NilValue{}, fmt.Errorf("Fs.move failed: %w", err)
		}
		return BoolValue{Val: true}, nil
	}

	copyFn := func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("Fs.copy requires src and dest path")
		}
		src, dest := args[0].String(), args[1].String()
		if err := copyPath(src, dest); err != nil {
			return NilValue{}, fmt.Errorf("Fs.copy failed: %w", err)
		}
		return BoolValue{Val: true}, nil
	}

	existsFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return BoolValue{Val: false}, nil
		}
		_, err := os.Lstat(args[0].String())
		return BoolValue{Val: err == nil}, nil
	}

	readFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Fs.read requires file path")
		}
		target := args[0].String()
		data, err := os.ReadFile(target)
		if err != nil {
			return NilValue{}, fmt.Errorf("Fs.read failed: %w", err)
		}
		return StringValue{Val: string(data)}, nil
	}

	writeFn := func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("Fs.write requires file path and content")
		}
		target := args[0].String()
		content := args[1].String()
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return NilValue{}, fmt.Errorf("Fs.write failed to create directory: %w", err)
		}
		if err := os.WriteFile(target, []byte(content), 0644); err != nil {
			return NilValue{}, fmt.Errorf("Fs.write failed: %w", err)
		}
		return BoolValue{Val: true}, nil
	}

	isDirFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Fs.isDir requires target path")
		}
		info, err := os.Stat(args[0].String())
		if err != nil {
			if os.IsNotExist(err) {
				return BoolValue{Val: false}, nil
			}
			return NilValue{}, fmt.Errorf("Fs.isDir failed: %w", err)
		}
		return BoolValue{Val: info.IsDir()}, nil
	}

	isFileFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Fs.isFile requires target path")
		}
		info, err := os.Stat(args[0].String())
		if err != nil {
			if os.IsNotExist(err) {
				return BoolValue{Val: false}, nil
			}
			return NilValue{}, fmt.Errorf("Fs.isFile failed: %w", err)
		}
		return BoolValue{Val: !info.IsDir()}, nil
	}

	sizeFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Fs.size requires file path")
		}
		info, err := os.Stat(args[0].String())
		if err != nil {
			return NilValue{}, fmt.Errorf("Fs.size failed: %w", err)
		}
		return NumberValue{Val: float64(info.Size())}, nil
	}

	return &UnitInstance{
		UnitName:   "Fs",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"EnsureDir": ensureDirFn,
			"ensureDir": ensureDirFn,
			"ensuredir": ensureDirFn,
			"Remove":    removeFn,
			"remove":    removeFn,
			"Move":      moveFn,
			"move":      moveFn,
			"Copy":      copyFn,
			"copy":      copyFn,
			"Exists":    existsFn,
			"exists":    existsFn,
			"Read":      readFn,
			"read":      readFn,
			"Write":     writeFn,
			"write":     writeFn,
			"IsDir":     isDirFn,
			"isDir":     isDirFn,
			"isdir":     isDirFn,
			"IsFile":    isFileFn,
			"isFile":    isFileFn,
			"isfile":    isFileFn,
			"Size":      sizeFn,
			"size":      sizeFn,
		},
		Directives: make(map[string]NativeDirective),
	}
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		s := filepath.Join(src, entry.Name())
		d := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
		} else {
			if err := copyFile(s, d); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractZip(zipBytes []byte, destDir string) error {
	r, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return err
	}
	cleanDest := filepath.Clean(destDir)
	for _, f := range r.File {
		targetPath := filepath.Join(destDir, f.Name)
		cleanTarget := filepath.Clean(targetPath)
		if !strings.HasPrefix(cleanTarget, cleanDest) {
			continue // Zip slip mitigation
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(cleanTarget, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		outFile, err := os.OpenFile(cleanTarget, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec < 1024 {
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	} else if bytesPerSec < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024.0)
	} else {
		return fmt.Sprintf("%.1f MB/s", bytesPerSec/(1024.0*1024.0))
	}
}
