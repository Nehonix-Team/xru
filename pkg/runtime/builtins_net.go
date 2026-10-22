package runtime

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func newNetUnit() *UnitInstance {
	var netUnit *UnitInstance

	fetchFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Net.Fetch requires url")
		}
		url := args[0].String()
		if len(args) >= 2 {
			dest := args[1].String()
			resp, err := http.Get(url)
			if err != nil {
				return BoolValue{Val: false}, err
			}
			defer resp.Body.Close()

			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return BoolValue{Val: false}, err
			}

			out, err := os.Create(dest)
			if err != nil {
				return BoolValue{Val: false}, err
			}
			defer out.Close()

			_, err = io.Copy(out, resp.Body)
			return BoolValue{Val: err == nil}, err
		}

		resp, err := http.Get(url)
		if err != nil {
			return NilValue{}, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return NilValue{}, fmt.Errorf("HTTP error: %s", resp.Status)
		}
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return NilValue{}, err
		}
		return StringValue{Val: string(bodyBytes)}, nil
	}

	cloneFn := func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("Net.Clone requires repo url and dest")
		}
		repo := args[0].String()
		dest := args[1].String()
		if !filepath.IsAbs(dest) && netUnit != nil && netUnit.WorkingDir != "" {
			dest = filepath.Join(netUnit.WorkingDir, dest)
		}

		cmd := exec.Command("git", "clone", "--depth", "1", repo, dest)
		if netUnit != nil && netUnit.WorkingDir != "" {
			cmd.Dir = netUnit.WorkingDir
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		return BoolValue{Val: err == nil}, err
	}

	netUnit = &UnitInstance{
		UnitName:   "Net",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Fetch": fetchFn,
			"fetch": fetchFn,
			"Clone": cloneFn,
			"clone": cloneFn,
		},
		Directives: map[string]NativeDirective{},
	}
	return netUnit
}
