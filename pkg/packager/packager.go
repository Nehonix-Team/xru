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

		// Ne JAMAIS empaqueter le fichier d'output en cours d'écriture ni son dossier parent out/
		absPath, _ := filepath.Abs(path)
		absOut, _ := filepath.Abs(outputFile)
		if absPath == absOut || strings.HasPrefix(relPath, "out/") || relPath == "out" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
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

		// Si c'est un lien symbolique ou un dossier, on ne fait pas de io.Copy de fichier régulier
		realInfo, statErr := os.Stat(path)
		if statErr == nil && realInfo.IsDir() {
			header.Typeflag = tar.TypeDir
			header.Name += "/"
			return tw.WriteHeader(header)
		}

		if info.Mode().IsRegular() {
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
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

func (p *Packager) isIgnored(relPath string, patterns []string) bool {
	cleanPath := filepath.ToSlash(relPath)
	baseName := filepath.Base(cleanPath)

	for _, pat := range patterns {
		pat = filepath.ToSlash(strings.TrimSpace(pat))
		if pat == "" || strings.HasPrefix(pat, "#") {
			continue
		}

		// 1. Dossier ciblé explicitement avec slash terminal (ex: "dist/", "node_modules/")
		if strings.HasSuffix(pat, "/") {
			dirPat := strings.TrimSuffix(pat, "/")
			if cleanPath == dirPat || strings.HasPrefix(cleanPath, dirPat+"/") {
				return true
			}
			continue
		}

		// 2. Correspondance exacte sur chemin complet ou segment (ex: "README.md", ".git")
		if cleanPath == pat {
			return true
		}
		// Si le motif est un nom de dossier racine/intermédiaire (ex: "dist", "tests")
		if strings.HasPrefix(cleanPath, pat+"/") {
			return true
		}

		// 3. Glob matching sur le nom du fichier (ex: "*.log", "*.zip", "TARGET_*.md")
		if matched, _ := filepath.Match(pat, baseName); matched {
			return true
		}

		// 4. Glob matching sur le chemin relatif complet (ex: "tests/*", "docs/*.md")
		if matched, _ := filepath.Match(pat, cleanPath); matched {
			return true
		}
	}
	return false
}
