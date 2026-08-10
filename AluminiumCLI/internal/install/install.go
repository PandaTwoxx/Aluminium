package install

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PandaTwoxx/Aluminium/internal/client"
	"github.com/PandaTwoxx/Aluminium/internal/config"
	"github.com/PandaTwoxx/Aluminium/internal/graph"
	"github.com/PandaTwoxx/Aluminium/internal/prompt"
)

type InstalledPackage struct {
	Version string `json:"version"`
	Server  string `json:"server"`
}

type InstalledState struct {
	Packages map[string]InstalledPackage `json:"packages"`
}

func GetInstalledStateFilePath() (string, error) {
	dir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "installed.json"), nil
}

func LoadInstalledState() (*InstalledState, error) {
	path, err := GetInstalledStateFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &InstalledState{Packages: make(map[string]InstalledPackage)}, nil
		}
		return nil, err
	}
	var state InstalledState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	if state.Packages == nil {
		state.Packages = make(map[string]InstalledPackage)
	}
	return &state, nil
}

func SaveInstalledState(state *InstalledState) error {
	path, err := GetInstalledStateFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

var envVarOrder = []string{
	"PATH",
	"LD_LIBRARY_PATH",
	"DYLD_LIBRARY_PATH",
	"CPATH",
	"PKG_CONFIG_PATH",
}

func envVarsForInstallDir(rel string) []string {
	rel = filepath.ToSlash(rel)
	switch rel {
	case "bin", "sbin":
		return []string{"PATH"}
	case "lib", "lib64":
		return []string{"LD_LIBRARY_PATH", "DYLD_LIBRARY_PATH"}
	case "include":
		return []string{"CPATH"}
	case "lib/pkgconfig", "lib64/pkgconfig", "share/pkgconfig":
		return []string{"PKG_CONFIG_PATH"}
	}

	parts := strings.Split(rel, "/")
	if len(parts) == 0 {
		return nil
	}

	switch parts[len(parts)-1] {
	case "bin", "sbin":
		return []string{"PATH"}
	case "lib", "lib64":
		return []string{"LD_LIBRARY_PATH", "DYLD_LIBRARY_PATH"}
	case "include":
		return []string{"CPATH"}
	case "pkgconfig":
		if len(parts) >= 2 {
			parent := parts[len(parts)-2]
			if parent == "lib" || parent == "lib64" || (len(parts) == 2 && parts[0] == "share") {
				return []string{"PKG_CONFIG_PATH"}
			}
		}
	}

	return nil
}

func hasInstallLayout(dir string) bool {
	for _, sub := range []string{"bin", "lib", "lib64", "include", "sbin"} {
		info, err := os.Stat(filepath.Join(dir, sub))
		if err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

// normalizeExtractedLayout hoists a single top-level directory from tarballs
// such as mypkg-1.0/bin/... into install/mypkg/bin/...
func normalizeExtractedLayout(destDir string) error {
	if hasInstallLayout(destDir) {
		return nil
	}

	entries, err := os.ReadDir(destDir)
	if err != nil {
		return err
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		return nil
	}

	nested := filepath.Join(destDir, entries[0].Name())
	if !hasInstallLayout(nested) {
		return nil
	}

	innerEntries, err := os.ReadDir(nested)
	if err != nil {
		return err
	}
	for _, entry := range innerEntries {
		if err := os.Rename(filepath.Join(nested, entry.Name()), filepath.Join(destDir, entry.Name())); err != nil {
			return fmt.Errorf("failed to normalize install layout: %w", err)
		}
	}
	return os.Remove(nested)
}

func collectEnvPathsFromPackage(pkgDir string) map[string][]string {
	paths := map[string][]string{
		"PATH":              {},
		"LD_LIBRARY_PATH":   {},
		"DYLD_LIBRARY_PATH": {},
		"CPATH":             {},
		"PKG_CONFIG_PATH":   {},
	}

	_ = filepath.WalkDir(pkgDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(pkgDir, path)
		if err != nil || rel == "." {
			return nil
		}

		for _, envVar := range envVarsForInstallDir(rel) {
			paths[envVar] = append(paths[envVar], path)
		}
		return nil
	})

	return paths
}

// RegenerateEnvFile scans all installed package directories and writes
// ~/.aluminium/env with exports for PATH, LD_LIBRARY_PATH, DYLD_LIBRARY_PATH,
// CPATH and PKG_CONFIG_PATH based on any bin/, lib/, include/, lib/pkgconfig/
// subdirectories that exist.
func RegenerateEnvFile(state *InstalledState) error {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return err
	}
	installBase := filepath.Join(configDir, "install")

	collected := map[string][]string{
		"PATH":              {},
		"LD_LIBRARY_PATH":   {},
		"DYLD_LIBRARY_PATH": {},
		"CPATH":             {},
		"PKG_CONFIG_PATH":   {},
	}

	for pkgName := range state.Packages {
		pkgDir := filepath.Join(installBase, pkgName)
		for envVar, pkgPaths := range collectEnvPathsFromPackage(pkgDir) {
			collected[envVar] = append(collected[envVar], pkgPaths...)
		}
	}

	var sb strings.Builder
	sb.WriteString("# Aluminium package manager environment\n")
	sb.WriteString("# Auto-generated — do not edit manually.\n")
	sb.WriteString("# Source this file in your shell config: source ~/.aluminium/env\n\n")

	for _, envVar := range envVarOrder {
		paths := collected[envVar]
		if len(paths) == 0 {
			continue
		}
		joined := strings.Join(paths, ":")
		sb.WriteString(fmt.Sprintf("export %s=\"%s:$%s\"\n", envVar, joined, envVar))
	}

	envPath := filepath.Join(configDir, "env")
	if err := os.WriteFile(envPath, []byte(sb.String()), 0644); err != nil {
		return err
	}

	return nil
}

// EnsureShellSourced checks the user's shell config file (~/.zshrc or
// ~/.bash_profile) for the `source ~/.aluminium/env` line and appends it
// if it is missing, printing a message to the user either way.
func EnsureShellSourced() {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return
	}
	envPath := filepath.Join(configDir, "env")
	sourceLine := fmt.Sprintf("source \"%s\"", envPath)

	// Determine which shell config to use
	shell := os.Getenv("SHELL")
	var rcFile string
	switch {
	case strings.Contains(shell, "zsh"):
		rcFile = filepath.Join(os.Getenv("HOME"), ".zshrc")
	case strings.Contains(shell, "bash"):
		rcFile = filepath.Join(os.Getenv("HOME"), ".bash_profile")
	default:
		fmt.Printf("\nTo apply environment changes, add the following to your shell config:\n  %s\n", sourceLine)
		return
	}

	data, err := os.ReadFile(rcFile)
	if err == nil && strings.Contains(string(data), sourceLine) {
		// Already sourced — silently update env file only
		return
	}

	// Append the source line
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Printf("\nTo apply environment changes, add the following to your shell config:\n  %s\n", sourceLine)
		return
	}
	defer f.Close()

	_, _ = fmt.Fprintf(f, "\n# Added by Aluminium package manager\n%s\n", sourceLine)
	fmt.Printf("✓ Added '%s' to %s\n", sourceLine, rcFile)
	fmt.Println("  Run `source " + rcFile + "` or open a new terminal to apply changes.")
}

func confirmBuildFromSource(cfg *config.Config, pkgName string) (bool, error) {
	msg := fmt.Sprintf("Prebuilt binary for package %s not found. Do you want to build it from source?", pkgName)
	if prompt.FromConfig(cfg) {
		return prompt.Confirm(msg, false)
	}
	return prompt.SimpleConfirm(msg, true)
}

func checkBuildSystem(buildSystem string) error {
	switch buildSystem {
	case "cmake":
		if _, err := exec.LookPath("cmake"); err != nil {
			return fmt.Errorf("Build system cmake is not installed. Please install cmake first in order to build this package.")
		}
	case "make":
		if _, err := exec.LookPath("make"); err != nil {
			return fmt.Errorf("Build system make is not installed. Please install make first in order to build this package.")
		}
	case "meson":
		if _, err := exec.LookPath("meson"); err != nil {
			return fmt.Errorf("Build system meson is not installed. Please install meson first in order to build this package.")
		}
	}
	return nil
}

func runScript(scriptContent, workingDir string) error {
	scriptPath := filepath.Join(workingDir, "run_setup.sh")
	err := os.WriteFile(scriptPath, []byte("#!/bin/bash\n"+scriptContent+"\n"), 0755)
	if err != nil {
		return err
	}
	defer os.Remove(scriptPath)

	cmd := exec.Command("bash", "run_setup.sh")
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func extractTarGz(gzipStream io.Reader, destDir string) error {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return err
	}
	defer uncompressedStream.Close()

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(filepath.Separator)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

func isArchiveSourceURL(sourceURL string) bool {
	lower := strings.ToLower(sourceURL)
	return strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") || strings.HasSuffix(lower, ".tar")
}

func downloadAndExtractSourceArchive(sourceURL, destDir string) error {
	resp, err := http.Get(sourceURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("source download failed with status %s", resp.Status)
	}

	tempFile, err := os.CreateTemp("", "aluminium-source-*.tar.gz")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		return err
	}
	if _, err := tempFile.Seek(0, 0); err != nil {
		return err
	}

	file, err := os.Open(tempPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return extractTarGz(file, destDir)
}

func prepareSourceWorkspace(sourceURL, workspaceDir string) (string, error) {
	if isArchiveSourceURL(sourceURL) {
		if err := downloadAndExtractSourceArchive(sourceURL, workspaceDir); err != nil {
			return "", err
		}
		entries, err := os.ReadDir(workspaceDir)
		if err != nil {
			return "", err
		}
		if len(entries) == 1 && entries[0].IsDir() {
			return filepath.Join(workspaceDir, entries[0].Name()), nil
		}
		return workspaceDir, nil
	}

	cmd := exec.Command("git", "clone", sourceURL, workspaceDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return workspaceDir, nil
}

func InstallSinglePackage(node *graph.Node, api *client.APIClient, cfg *config.Config, state *InstalledState) error {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return err
	}
	destDir := filepath.Join(configDir, "install", node.Name)

	// Step 1: Try prebuilt
	fmt.Printf("Attempting to download prebuilt binary for %s@%s...\n", node.Name, node.Version)
	token := cfg.Servers[node.ServerURL].Token
	stream, err := api.DownloadPrebuilt(node.ServerURL, node.Name, node.Version, token)
	if err == nil && stream != nil {
		defer stream.Close()
		fmt.Printf("Extracting prebuilt binary to %s...\n", destDir)

		os.RemoveAll(destDir)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create install directory: %w", err)
		}

		if err := extractTarGz(stream, destDir); err != nil {
			return fmt.Errorf("failed to extract prebuilt binary: %w", err)
		}
		if err := normalizeExtractedLayout(destDir); err != nil {
			return fmt.Errorf("failed to normalize prebuilt install layout: %w", err)
		}

		state.Packages[node.Name] = InstalledPackage{
			Version: node.Version,
			Server:  node.ServerURL,
		}
		if err := SaveInstalledState(state); err != nil {
			return err
		}
		if err := RegenerateEnvFile(state); err != nil {
			fmt.Printf("Warning: could not update env file: %v\n", err)
		}
		fmt.Printf("Successfully installed prebuilt package %s@%s.\n", node.Name, node.Version)
		return nil
	}

	// Step 2: Fallback to Build from Source
	fmt.Printf("Prebuilt binary download failed or is not available for %s@%s. (Error: %v)\n", node.Name, node.Version, err)
	if node.BuildSetup == nil {
		return fmt.Errorf("no prebuilt binary or build script available for package %s", node.Name)
	}

	proceed, askErr := confirmBuildFromSource(cfg, node.Name)
	if askErr != nil {
		return askErr
	}
	if !proceed {
		return fmt.Errorf("installation aborted by user")
	}

	if err := checkBuildSystem(node.BuildSystem); err != nil {
		return err
	}

	buildDir := filepath.Join(configDir, "build", node.Name)
	os.RemoveAll(buildDir)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build workspace: %w", err)
	}
	defer os.RemoveAll(buildDir)

	workingDir := buildDir
	if node.BuildSetup.SourceCodeURL != "" {
		fmt.Printf("Fetching source code from %s...\n", node.BuildSetup.SourceCodeURL)
		preparedDir, err := prepareSourceWorkspace(node.BuildSetup.SourceCodeURL, buildDir)
		if err != nil {
			fmt.Printf("Warning: failed to prepare source workspace: %v. Proceeding to run build script in workspace.\n", err)
		} else {
			workingDir = preparedDir
		}
	}

	if node.BuildSetup.BuildScript != "" {
		fmt.Println("Running build script...")
		if err := runScript(node.BuildSetup.BuildScript, workingDir); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}
	}

	if node.BuildSetup.InstallScript != "" {
		fmt.Println("Running install script...")
		if err := runScript(node.BuildSetup.InstallScript, workingDir); err != nil {
			return fmt.Errorf("install failed: %w", err)
		}
	}

	state.Packages[node.Name] = InstalledPackage{
		Version: node.Version,
		Server:  node.ServerURL,
	}
	if err := SaveInstalledState(state); err != nil {
		return err
	}
	if err := RegenerateEnvFile(state); err != nil {
		fmt.Printf("Warning: could not update env file: %v\n", err)
	}

	fmt.Printf("Successfully built and installed package %s@%s from source.\n", node.Name, node.Version)
	return nil
}
