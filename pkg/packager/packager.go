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

	"github.com/Nehonix-Team/xru/pkg/ast"
)

const (
	DefaultIgnoreFile  = ".xruignore"
	DefaultIncludeFile = ".xruinclude"
)

type PackageManifest struct {
	Version       string            `json:"version"`
	EntryDir      string            `json:"entryDir"`
	Args          []string          `json:"args,omitempty"`
	IgnoredPaths  []string          `json:"ignoredPaths,omitempty"`
	IncludedPaths []string          `json:"includedPaths,omitempty"`
	Metadata      map[string]string `json:"metadata"`
}

type Packager struct {
	rootDir     string
	ignoreFile  string
	includeFile string
	isWhitelist bool
	Verbose     bool
}

func New(rootDir string, filterFile string) *Packager {
	pkg := &Packager{
		rootDir:     rootDir,
		ignoreFile:  DefaultIgnoreFile,
		includeFile: DefaultIncludeFile,
		Verbose:     false,
	}

	if filterFile == DefaultIncludeFile {
		pkg.isWhitelist = true
	} else if filterFile != "" && filterFile != DefaultIgnoreFile {
		pkg.ignoreFile = filterFile
	}

	// Auto-détection : si .xruinclude existe dans rootDir, il prend automatiquement la priorité sur .xruignore
	if _, err := os.Stat(filepath.Join(rootDir, pkg.includeFile)); err == nil {
		pkg.isWhitelist = true
	}

	return pkg
}

func (p *Packager) FilterFile() string {
	if p.isWhitelist {
		return p.includeFile
	}
	return p.ignoreFile
}

func (p *Packager) IsWhitelist() bool {
	return p.isWhitelist
}

// Pack compresse l'arborescence dans un bundle binaire compartimenté :
// - manifest.json (métadonnées & arguments extraits)
// - core/         (template canonique dédoublonné SANS les fichiers .xru)
// - rules/        (tous les descripteurs *.rules.xru + main.xru)
func (p *Packager) Pack(outputFile string, entryDir string) error {
	var filterPatterns []string
	if p.isWhitelist {
		filterPatterns = p.loadPatterns(p.includeFile)
	} else {
		filterPatterns = p.loadPatterns(p.ignoreFile)
	}

	outFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// 1. Bundling de toutes les règles et modules en un AST complet
	bundledProg, err := p.BuildBundledProgram()
	if err != nil {
		return fmt.Errorf("failed to bundle modules: %w", err)
	}

	// Extraire les flags arg déclarés pour le manifest
	var detectedArgs []string
	if bundledProg != nil {
		for _, stmt := range bundledProg.Statements {
			if argStmt, ok := stmt.(*ast.ArgStatement); ok {
				detectedArgs = append(detectedArgs, argStmt.String())
			}
		}
	}

	// Écriture du manifest.json à la racine de l'archive
	manifest := PackageManifest{
		Version:  "2.0.0",
		EntryDir: "core",
		Args:     detectedArgs,
		Metadata: map[string]string{
			"generator": "xru-pack-v2",
		},
	}
	if p.isWhitelist {
		manifest.IncludedPaths = filterPatterns
	} else {
		manifest.IgnoredPaths = filterPatterns
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

	// 2. Minification unifiée : main.xru à la racine de l'archive
	if bundledProg != nil {
		mangler := NewMangler(bundledProg)
		bundledContent := mangler.Minify(bundledProg)
		if bundledContent != "" {
			mainHdr := &tar.Header{
				Name: "main.xru",
				Mode: 0644,
				Size: int64(len(bundledContent)),
			}
			if err := tw.WriteHeader(mainHdr); err != nil {
				return err
			}
			if _, err := tw.Write([]byte(bundledContent)); err != nil {
				return err
			}
		}
	}

	// 3. Parcourt et inclusion du template : core/ (sans duplication de fichiers .xru)
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

		// Ni .xruignore ni .xruinclude ne doivent JAMAIS être empaquetés
		if relPath == ".xruignore" || filepath.Base(relPath) == ".xruignore" ||
			relPath == ".xruinclude" || filepath.Base(relPath) == ".xruinclude" {
			if p.Verbose {
				fmt.Printf("  \033[90m[EXCLUDE]\033[0m %s\n", relPath)
			}
			return nil
		}

		// Ne pas empaqueter .git
		if relPath == ".git" || strings.HasPrefix(relPath, ".git/") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		cleanRel := filepath.ToSlash(relPath)
		baseName := filepath.Base(cleanRel)

		// Les fichiers .xru sont déjà assemblés et minifiés dans main.xru
		if strings.HasSuffix(baseName, ".xru") {
			return nil
		}

		if p.isWhitelist {
			if info.IsDir() {
				if !p.canDirContainIncluded(cleanRel, filterPatterns) {
					if p.Verbose {
						fmt.Printf("  \033[90m[EXCLUDE DIR]\033[0m %s\n", relPath)
					}
					return filepath.SkipDir
				}
				return nil
			}

			if !p.isIncluded(cleanRel, filterPatterns) {
				if p.Verbose {
					fmt.Printf("  \033[90m[EXCLUDE]\033[0m %s\n", relPath)
				}
				return nil
			}
		} else {
			if p.isIgnored(cleanRel, filterPatterns) {
				if p.Verbose {
					fmt.Printf("  \033[90m[EXCLUDE]\033[0m %s\n", relPath)
				}
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if info.IsDir() {
				return nil
			}
		}

		targetTarPath := "core/" + cleanRel

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
						if p.isWhitelist {
							if subInfo.IsDir() {
								if !p.canDirContainIncluded(nestedRel, filterPatterns) {
									return filepath.SkipDir
								}
								return nil
							}
							if !p.isIncluded(nestedRel, filterPatterns) {
								return nil
							}
						} else {
							if p.isIgnored(nestedRel, filterPatterns) {
								if subInfo.IsDir() {
									return filepath.SkipDir
								}
								return nil
							}
						}
						if subInfo.Mode().IsRegular() {
							if strings.HasSuffix(filepath.Base(nestedRel), ".xru") {
								return nil
							}
							nestedTarPath := "core/" + nestedRel
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

func (p *Packager) loadPatterns(filterFile string) []string {
	var list []string
	content, err := os.ReadFile(filepath.Join(p.rootDir, filterFile))
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
	cleanPath := filepath.ToSlash(filepath.Clean(relPath))
	baseName := filepath.Base(cleanPath)

	for _, rawPat := range patterns {
		pat := filepath.ToSlash(strings.TrimSpace(rawPat))
		pat = strings.TrimPrefix(pat, "./")
		if pat == "" || strings.HasPrefix(pat, "#") {
			continue
		}

		// 1. Dossier ciblé explicitement avec wildcard ou slash (ex: "mods/*", "mods/**", "dist/", "data/")
		if strings.HasSuffix(pat, "/*") || strings.HasSuffix(pat, "/**") || strings.HasSuffix(pat, "/") {
			dirPat := pat
			dirPat = strings.TrimSuffix(dirPat, "/**")
			dirPat = strings.TrimSuffix(dirPat, "/*")
			dirPat = strings.TrimSuffix(dirPat, "/")
			if cleanPath == dirPat || strings.HasPrefix(cleanPath, dirPat+"/") {
				return true
			}
			continue
		}

		// 2. Correspondance exacte sur chemin complet ou segment (ex: "README.md", ".git", "dist")
		if cleanPath == pat || strings.HasPrefix(cleanPath, pat+"/") {
			return true
		}

		// 3. Glob matching sur le nom du fichier (ex: "*.log", "*.zip")
		if matched, _ := filepath.Match(pat, baseName); matched {
			return true
		}

		// 4. Glob matching avec sous-dossier (ex: "data/*.xems", "src/*.tmp")
		if strings.Contains(pat, "/") {
			dirPart := filepath.Dir(pat)
			filePart := filepath.Base(pat)
			if filepath.Dir(cleanPath) == dirPart {
				if matched, _ := filepath.Match(filePart, baseName); matched {
					return true
				}
			}
		}

		// 5. Glob matching sur le chemin relatif complet (ex: "tests/*", "docs/*.md")
		if matched, _ := filepath.Match(pat, cleanPath); matched {
			return true
		}
	}
	return false
}

func (p *Packager) isIncluded(relPath string, patterns []string) bool {
	cleanPath := filepath.ToSlash(filepath.Clean(relPath))
	baseName := filepath.Base(cleanPath)

	for _, rawPat := range patterns {
		pat := filepath.ToSlash(strings.TrimSpace(rawPat))
		pat = strings.TrimPrefix(pat, "./")
		if pat == "" || strings.HasPrefix(pat, "#") {
			continue
		}

		hasWildcard := strings.ContainsAny(pat, "*?[")

		// 1. Dossier ciblé explicitement avec wildcard ou slash (ex: "src/", "core/**", "configs/*")
		if strings.HasSuffix(pat, "/*") || strings.HasSuffix(pat, "/**") || strings.HasSuffix(pat, "/") {
			dirPat := pat
			dirPat = strings.TrimSuffix(dirPat, "/**")
			dirPat = strings.TrimSuffix(dirPat, "/*")
			dirPat = strings.TrimSuffix(dirPat, "/")
			if cleanPath == dirPat || strings.HasPrefix(cleanPath, dirPat+"/") {
				return true
			}
			continue
		}

		// 2. Motif avec wildcard (ex: "*.json", "src/*.ts", "**/*.ts")
		if hasWildcard {
			if strings.Contains(pat, "/") {
				dirPart := filepath.Dir(pat)
				filePart := filepath.Base(pat)
				if filepath.Dir(cleanPath) == dirPart {
					if matched, _ := filepath.Match(filePart, baseName); matched {
						return true
					}
				}
				if matched, _ := filepath.Match(pat, cleanPath); matched {
					return true
				}
			} else {
				// Glob sans slash (ex: "*.json") -> s'applique au nom de fichier
				if matched, _ := filepath.Match(pat, baseName); matched {
					return true
				}
			}
			continue
		}

		// 3. Fichier ou chemin littéral sans wildcard (ex: "package.json", "scripts/meta.json")
		if cleanPath == pat {
			return true
		}
	}
	return false
}

func (p *Packager) canDirContainIncluded(relDir string, patterns []string) bool {
	cleanDir := filepath.ToSlash(filepath.Clean(relDir))

	for _, rawPat := range patterns {
		pat := filepath.ToSlash(strings.TrimSpace(rawPat))
		pat = strings.TrimPrefix(pat, "./")
		if pat == "" || strings.HasPrefix(pat, "#") {
			continue
		}

		hasWildcard := strings.ContainsAny(pat, "*?[")

		// 1. Dossier explicite (ex: "src/", "src/**")
		if strings.HasSuffix(pat, "/*") || strings.HasSuffix(pat, "/**") || strings.HasSuffix(pat, "/") {
			dirPat := pat
			dirPat = strings.TrimSuffix(dirPat, "/**")
			dirPat = strings.TrimSuffix(dirPat, "/*")
			dirPat = strings.TrimSuffix(dirPat, "/")
			if cleanDir == dirPat || strings.HasPrefix(cleanDir, dirPat+"/") || strings.HasPrefix(dirPat, cleanDir+"/") {
				return true
			}
			continue
		}

		// 2. Motif avec wildcard
		if hasWildcard {
			if strings.Contains(pat, "/") {
				dirPart := filepath.Dir(pat)
				if strings.HasPrefix(dirPart, cleanDir) || cleanDir == dirPart || strings.HasPrefix(cleanDir, dirPart+"/") {
					return true
				}
			} else {
				// Un glob général comme "*.ts" peut se trouver dans n'importe quel dossier
				return true
			}
			continue
		}

		// 3. Fichier littéral avec chemin relatif (ex: "scripts/meta.json")
		if strings.Contains(pat, "/") {
			dirPart := filepath.Dir(pat)
			if strings.HasPrefix(dirPart, cleanDir) || cleanDir == dirPart || strings.HasPrefix(cleanDir, dirPart+"/") {
				return true
			}
			continue
		}

		// 4. Fichier littéral racine (ex: "package.json", "tsconfig.json")
		// Ne peut PAS être contenu dans un sous-dossier, donc on ne traverse pas de sous-dossier pour lui
	}
	return false
}
