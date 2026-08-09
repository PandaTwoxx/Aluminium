package container

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/PandaTwoxx/Aluminium/internal/config"
	"github.com/PandaTwoxx/Aluminium/internal/install"
)

// IsToolAvailable checks if a binary tool is available in PATH or in Aluminium's install directory.
func IsToolAvailable(toolName string) bool {
	if _, err := exec.LookPath(toolName); err == nil {
		return true
	}

	// Check Aluminium install directory ~/.aluminium/install/<toolName>/bin/<toolName>
	if cfgDir, err := config.GetConfigDir(); err == nil {
		binPath := filepath.Join(cfgDir, "install", toolName, "bin", toolName)
		if info, err := os.Stat(binPath); err == nil && !info.IsDir() {
			return true
		}
	}

	// Check installed state
	if state, err := install.LoadInstalledState(); err == nil {
		if _, ok := state.Packages[toolName]; ok {
			return true
		}
	}

	return false
}

// GetToolPath returns the executable path for a tool (either system PATH or Aluminium bin).
func GetToolPath(toolName string) string {
	if path, err := exec.LookPath(toolName); err == nil {
		return path
	}
	if cfgDir, err := config.GetConfigDir(); err == nil {
		binPath := filepath.Join(cfgDir, "install", toolName, "bin", toolName)
		if info, err := os.Stat(binPath); err == nil && !info.IsDir() {
			return binPath
		}
	}
	return toolName
}

// EnsureContainerTools checks if colima and nerdctl are installed.
// If missing, returns a descriptive error suggesting `aluminium install colima nerdctl`.
func EnsureContainerTools() error {
	var missing []string

	if !IsToolAvailable("colima") {
		missing = append(missing, "colima")
	}
	if !IsToolAvailable("nerdctl") {
		missing = append(missing, "nerdctl")
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"missing container management dependencies (%s).\nPlease install them by running: aluminium install %s",
			strings.Join(missing, ", "),
			strings.Join(missing, " "),
		)
	}

	return nil
}

// EnsureRuntimeRunning ensures that colima daemon is running.
func EnsureRuntimeRunning() error {
	if err := EnsureContainerTools(); err != nil {
		return err
	}

	colimaBin := GetToolPath("colima")

	// Check colima status
	cmd := exec.Command(colimaBin, "status")
	output, err := cmd.CombinedOutput()
	if err == nil && strings.Contains(strings.ToLower(string(output)), "running") {
		return nil
	}

	// Colima not running, attempt to start it
	fmt.Println("Starting colima container runtime daemon...")
	startCmd := exec.Command(colimaBin, "start")
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr

	if err := startCmd.Run(); err != nil {
		// On Linux if containerd / nerdctl is natively available without colima, check nerdctl
		if runtime.GOOS == "linux" {
			nerdctlBin := GetToolPath("nerdctl")
			checkCmd := exec.Command(nerdctlBin, "info")
			if checkCmd.Run() == nil {
				return nil
			}
		}
		return fmt.Errorf("failed to start colima container runtime: %w\nMake sure colima and nerdctl are installed and configured properly", err)
	}

	return nil
}
