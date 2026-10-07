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

	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/parser"
)

type PackageManifest struct {
	Version      string            `json:"version"`
	EntryDir     string            `json:"entryDir"`
	Args         []string          `json:"args,omitempty"`
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

// Pack compresse l'arborescence dans un bundle binaire compartimenté :
// - manifest.json (métadonnées & arguments extraits)
// - core/         (template canonique dédoublonné SANS les fichiers .xru)
// - rules/        (tous les descripteurs *.rules.xru + main.xru)
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

	// 1. Extraire les flags #ARG déclarés dans main.xru pour le manifest
	var detectedArgs []string
	if mainContent, err := os.ReadFile(filepath.Join(p.rootDir, "main.xru")); err == nil {
		l := lexer.New("main.xru", string(mainContent))
		pr := parser.New(l)
		prog := pr.ParseProgram()
		for _, stmt := range prog.Statements {
			if strings.HasPrefix(stmt.String(), "#ARG") {
				detectedArgs = append(detectedArgs, stmt.String())
			}
		}
	}

	// Écriture du manifest.json à la racine de l'archive
	manifest := PackageManifest{
		Version:      "2.0.0",
		EntryDir:     "core",
		Args:         detectedArgs,
		IgnoredPaths: ignored,
		Metadata: map[string]string{
			"generator": "xru-pack-v2",
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

	// 2. Parcourt et inclusion compartimentée : core/ et rules/
	sourcePath := filepath.Join(p.rootDir, entryDir)
	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(p.rootDir, path)
		if err != nil || relPath == "." {
			return nil
		}

		// Ne JAMAIS empaqueter le fichier de sortie en cours d'écriture ni out/
		absPath, _ := filepath.Abs(path)
		absOut, _ := filepath.Abs(outputFile)
		if absPath == absOut || strings.HasPrefix(relPath, "out/") || relPath == "out" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Le fichier .xruignore lui-même ne doit JAMAIS être empaqueté
		if relPath == ".xruignore" || filepath.Base(relPath) == ".xruignore" {
			if p.Verbose {
				fmt.Printf("  \033[90m[EXCLUDE]\033[0m %s\n", relPath)
			}
			return nil
		}

		// Vérification d'exclusion par .xruignore
		if p.isIgnored(relPath, ignored) {
			if p.Verbose {
				fmt.Printf("  \033[90m[EXCLUDE]\033[0m %s\n", relPath)
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		cleanRel := filepath.ToSlash(relPath)
		baseName := filepath.Base(cleanRel)

		// Compartimentage : les fichiers .xru vont dans rules/, le reste dans core/
		var targetTarPath string
		isRuleFile := strings.HasSuffix(baseName, ".xru")

		if isRuleFile {
			if cleanRel == "main.xru" {
				targetTarPath = "rules/main.xru"
			} else {
				targetTarPath = "rules/" + cleanRel
			}
		} else {
			targetTarPath = "core/" + cleanRel
		}

		// Gestion des liens symboliques de dossiers
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(path)
			if err == nil {
				targetInfo, err := os.Stat(target)
				if err == nil && targetInfo.IsDir() {
					return filepath.Walk(target, func(subPath string, subInfo os.FileInfo, subErr error) error {
						if subErr != nil {
							return subErr
						}
						subRel, err := filepath.Rel(target, subPath)
						if err != nil || subRel == "." {
							return nil
						}
						nestedRel := filepath.ToSlash(filepath.Join(relPath, subRel))
						if p.isIgnored(nestedRel, ignored) {
							if subInfo.IsDir() {
								return filepath.SkipDir
							}
							return nil
						}
						if subInfo.Mode().IsRegular() {
							nestedTarPath := "core/" + nestedRel
							if strings.HasSuffix(filepath.Base(nestedRel), ".xru") {
								nestedTarPath = "rules/" + nestedRel
							}
							if p.Verbose {
								fmt.Printf("  \033[32m[ADD]\033[0m %s (%d bytes)\n", nestedTarPath, subInfo.Size())
							}
							subHdr, err := tar.FileInfoHeader(subInfo, subInfo.Name())
							if err != nil {
								return err
							}
							subHdr.Name = nestedTarPath
							if err := tw.WriteHeader(subHdr); err != nil {
								return err
							}
							f, err := os.Open(subPath)
							if err != nil {
								return err
							}
							defer f.Close()
							_, err = io.Copy(tw, f)
							return err
						}
						return nil
					})
				}
			}
		}

		if p.Verbose && !info.IsDir() {
			fmt.Printf("  \033[32m[ADD]\033[0m %s (%d bytes)\n", targetTarPath, info.Size())
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		header.Name = targetTarPath

		if info.IsDir() {
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

// Unpack extrait un package .xpack dans un dossier de destination en matérialisant core/
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

		// Lors du déballage, si c'est un fichier de "core/" ou "rules/", on le dépose à son emplacement dans le projet
		outRel := header.Name
		if strings.HasPrefix(outRel, "core/") {
			outRel = strings.TrimPrefix(outRel, "core/")
		} else if strings.HasPrefix(outRel, "rules/") {
			outRel = strings.TrimPrefix(outRel, "rules/")
		}

		targetPath := filepath.Join(destDir, outRel)
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
		return []string{}
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
		if strings.HasPrefix(cleanPath, pat+"/") {
			return true
		}

		// 3. Glob matching sur le nom du fichier (ex: "*.log", "*.zip")
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
