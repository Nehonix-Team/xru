package packager

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type PackageManifest struct {
	Version      string            `json:"version"`
	EntryDir     string            `json:"entryDir"`
	IgnoredPaths []string          `json:"ignoredPaths"`
	Metadata     map[string]string `json:"metadata"`
}

type Packager struct {
	rootDir    string
	ignoreFile string
	Verbose    bool
}

func New(rootDir string, ignoreFile string) *Packager {
	if ignoreFile == "" {
		ignoreFile = ".xruignore"
	}
	return &Packager{
		rootDir:    rootDir,
		ignoreFile: ignoreFile,
		Verbose:    false,
	}
}

// Pack compresse l'arborescence canonique nettoyée dans un bundle binaire structuré .xpack
func (p *Packager) Pack(outputFile string, entryDir string) error {
	ignored := p.loadIgnoredPatterns()

	outFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// 1. Écriture du manifest
	manifest := PackageManifest{
		Version:      "2.0.0",
		EntryDir:     entryDir,
		IgnoredPaths: ignored,
		Metadata: map[string]string{
			"generator": "xru-pack",
		},
	}
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	hdr := &tar.Header{
		Name: "manifest.json",
		Mode: 0644,
		Size: int64(len(manifestBytes)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(manifestBytes); err != nil {
		return err
	}

	// 2. Parcourt et inclusion de l'arborescence canonique
	sourcePath := filepath.Join(p.rootDir, entryDir)
	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(p.rootDir, path)
		if err != nil {
			return err
		}

		if p.isIgnored(relPath, ignored) {
			if p.Verbose {
				fmt.Printf("  \033[90m[EXCLUDE]\033[0m %s\n", relPath)
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if p.Verbose && !info.IsDir() {
			fmt.Printf("  \033[32m[ADD]\033[0m %s (%d bytes)\n", relPath, info.Size())
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(tw, file)
			return err
		}
		return nil
	})
}

// Unpack extrait un package .xpack dans un dossier de destination
func (p *Packager) Unpack(packFile string, destDir string) (*PackageManifest, error) {
	file, err := os.Open(packFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var manifest *PackageManifest

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if header.Name == "manifest.json" {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, err
			}
			var m PackageManifest
			if err := json.Unmarshal(data, &m); err == nil {
				manifest = &m
			}
			continue
		}

		targetPath := filepath.Join(destDir, header.Name)
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return nil, err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return nil, err
		}

		outFile, err := os.Create(targetPath)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(outFile, tr); err != nil {
			outFile.Close()
			return nil, err
		}
		outFile.Close()
	}

	return manifest, nil
}

func (p *Packager) loadIgnoredPatterns() []string {
	var list []string
	content, err := os.ReadFile(filepath.Join(p.rootDir, p.ignoreFile))
	if err != nil {
		return []string{} // Aucun fallback hardcodé : l'utilisateur pilote tout via son fichier ignore
	}
	lines := strings.Split(string(content), "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			list = append(list, trimmed)
		}
	}
	return list
}

func (p *Packager) isIgnored(path string, patterns []string) bool {
	for _, pat := range patterns {
		if strings.Contains(path, pat) {
			return true
		}
		matched, _ := filepath.Match(pat, filepath.Base(path))
		if matched {
			return true
		}
	}
	return false
}
