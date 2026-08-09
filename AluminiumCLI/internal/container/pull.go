package container

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/PandaTwoxx/Aluminium/internal/client"
)

// PullContainer downloads a container image from Aluminium server registry or Docker Hub and loads it into containerd.
func PullContainer(nameOrImage string, source string, server string, token string, api *client.APIClient) error {
	if err := EnsureRuntimeRunning(); err != nil {
		return err
	}

	nerdctlBin := GetToolPath("nerdctl")
	version := "latest"
	pkgName := nameOrImage

	if idx := strings.Index(nameOrImage, ":"); idx != -1 {
		pkgName = nameOrImage[:idx]
		version = nameOrImage[idx+1:]
	}

	if source == "aluminium" {
		fmt.Printf("Pulling container '%s@%s' from Aluminium server (%s)...\n", pkgName, version, server)

		stream, err := api.DownloadPrebuilt(server, pkgName, version, token)
		if err != nil {
			return fmt.Errorf("failed to download container from Aluminium server: %w", err)
		}
		defer stream.Close()

		tmpFile, err := os.CreateTemp("", "aluminium-pull-*.tar.gz")
		if err != nil {
			return fmt.Errorf("failed to create temporary pull file: %w", err)
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		if _, err := io.Copy(tmpFile, stream); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to save downloaded container payload: %w", err)
		}
		tmpFile.Close()

		fmt.Println("Loading container image into containerd via nerdctl...")
		loadCmd := exec.Command(nerdctlBin, "load", "-i", tmpPath)
		loadCmd.Stdout = os.Stdout
		loadCmd.Stderr = os.Stderr

		if err := loadCmd.Run(); err != nil {
			return fmt.Errorf("failed to load container image with nerdctl: %w", err)
		}

		fmt.Printf("✓ Successfully pulled and loaded container '%s@%s' from Aluminium server.\n", pkgName, version)
		return nil
	}

	// Default Docker Hub pull
	fmt.Printf("Pulling container image '%s' from Docker Hub...\n", nameOrImage)
	pullCmd := exec.Command(nerdctlBin, "pull", nameOrImage)
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr

	if err := pullCmd.Run(); err != nil {
		// Attempt Aluminium server fallback if dockerhub pull fails
		if server != "" {
			fmt.Printf("Docker Hub pull failed (%v). Attempting Aluminium server fallback...\n", err)
			return PullContainer(nameOrImage, "aluminium", server, token, api)
		}
		return fmt.Errorf("failed to pull image '%s': %w", nameOrImage, err)
	}

	// Tag image if simple name without repository was specified
	if !strings.Contains(nameOrImage, ":") {
		_ = exec.Command(nerdctlBin, "tag", nameOrImage+":latest", nameOrImage).Run()
	}

	fmt.Printf("✓ Successfully pulled image '%s'.\n", nameOrImage)
	return nil
}
