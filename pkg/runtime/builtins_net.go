package runtime

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

	portBusyCheck := func(portStr string, host string) bool {
		targetHost := "127.0.0.1"
		if host != "" {
			targetHost = host
		}
		// Try active TCP dial
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(targetHost, portStr), 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return true
		}
		// Try binding
		addr := ":" + portStr
		if host != "" {
			addr = net.JoinHostPort(host, portStr)
		}
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return true
		}
		_ = ln.Close()
		return false
	}

	parsePortVal := func(v Value) (string, error) {
		switch val := v.(type) {
		case NumberValue:
			p := int(val.Val)
			if p <= 0 || p > 65535 {
				return "", fmt.Errorf("invalid port number: %d", p)
			}
			return strconv.Itoa(p), nil
		default:
			s := strings.TrimSpace(v.String())
			p, err := strconv.Atoi(s)
			if err != nil || p <= 0 || p > 65535 {
				return "", fmt.Errorf("invalid port: %s", s)
			}
			return strconv.Itoa(p), nil
		}
	}

	isPortBusyFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Net.isPortBusy requires port")
		}
		portStr, err := parsePortVal(args[0])
		if err != nil {
			return NilValue{}, err
		}
		host := ""
		if len(args) >= 2 {
			host = strings.TrimSpace(args[1].String())
		}
		return BoolValue{Val: portBusyCheck(portStr, host)}, nil
	}

	isPortFreeFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Net.isPortFree requires port")
		}
		portStr, err := parsePortVal(args[0])
		if err != nil {
			return NilValue{}, err
		}
		host := ""
		if len(args) >= 2 {
			host = strings.TrimSpace(args[1].String())
		}
		return BoolValue{Val: !portBusyCheck(portStr, host)}, nil
	}

	netUnit = &UnitInstance{
		UnitName:   "Net",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Fetch":           fetchFn,
			"fetch":           fetchFn,
			"Clone":           cloneFn,
			"clone":           cloneFn,
			"isPortBusy":      isPortBusyFn,
			"IsPortBusy":      isPortBusyFn,
			"isPortFree":      isPortFreeFn,
			"IsPortFree":      isPortFreeFn,
			"isPortAvailable": isPortFreeFn,
			"IsPortAvailable": isPortFreeFn,
			"isPortInUse":     isPortBusyFn,
			"IsPortInUse":     isPortBusyFn,
		},
		Directives: map[string]NativeDirective{},
	}
	return netUnit
}
