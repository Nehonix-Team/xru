package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/Nehonix-Team/xru/pkg/engine"
)

// handleUpgrade télécharge et remplace le binaire xru par la dernière version.
func handleUpgrade() {
	binaryName := "xru"
	osName := runtime.GOOS
	archName := runtime.GOARCH
	ext := ""
	if osName == "windows" {
		ext = ".exe"
	}

	url := fmt.Sprintf(
		"https://github.com/Nehonix-Team/xru/releases/latest/download/%s-%s-%s%s",
		binaryName, osName, archName, ext,
	)

	executablePath, _ := os.Executable()
	tmpPath := executablePath + ".tmp"

	out, err := os.Create(tmpPath)
	if err != nil {
		fmt.Printf("%serror:%s could not create temp file: %v\n", engine.ColorRed, engine.ColorReset, err)
		os.Exit(1)
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("%serror:%s could not download upgrade: %v\n", engine.ColorRed, engine.ColorReset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	io.Copy(out, resp.Body)
	os.Chmod(tmpPath, 0755)
	os.Rename(tmpPath, executablePath)
}

