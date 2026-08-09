package container

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/PandaTwoxx/Aluminium/internal/client"
	"github.com/fatih/color"
)

// PublishContainer handles publishing a container setup / image to Aluminium server registry or Docker Hub.
func PublishContainer(spec *ContainerSpec, workDir string, server string, token string, api *client.APIClient) error {
	if err := EnsureRuntimeRunning(); err != nil {
		return err
	}

	if spec.Name == "" {
		return fmt.Errorf("container setup must specify a name (e.g. 'set name test')")
	}

	tag := fmt.Sprintf("%s:latest", spec.Name)

	// Check if local image already exists or build it
	nerdctlBin := GetToolPath("nerdctl")
	checkCmd := exec.Command(nerdctlBin, "image", "inspect", tag)
	if checkCmd.Run() != nil {
		// Image not found locally, build from spec
		builtTag, err := BuildContainerImage(spec, workDir)
		if err != nil {
			return fmt.Errorf("failed to build container image for publishing: %w", err)
		}
		tag = builtTag
	}

	if spec.Source == "dockerhub" {
		color.Cyan("Pushing container image '%s' to Docker Hub...\n", tag)
		cmd := exec.Command(nerdctlBin, "push", tag)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to push image to Docker Hub: %w", err)
		}
		color.Green("✓ Successfully pushed container image '%s' to Docker Hub.\n", tag)
		return nil
	}

	// Default / Aluminium server publish
	color.Cyan("Exporting container image '%s' for Aluminium server registry...\n", tag)
	tmpDir, err := os.MkdirTemp("", "aluminium-export-*")
	if err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tarPath := filepath.Join(tmpDir, fmt.Sprintf("%s.tar.gz", spec.Name))
	saveCmd := exec.Command(nerdctlBin, "save", tag, "-o", tarPath)
	saveCmd.Stdout = os.Stdout
	saveCmd.Stderr = os.Stderr
	if err := saveCmd.Run(); err != nil {
		return fmt.Errorf("failed to export container image tarball: %w", err)
	}

	version := "latest"
	color.Cyan("Registering container package '%s@%s' on Aluminium server (%s)...\n", spec.Name, version, server)

	regPayload := &client.RegisterPackagePayload{
		Name:         spec.Name,
		Version:      version,
		BuildSystem:  "none",
		Dependencies: spec.AluminiumPackages,
	}

	// Register package (ignore error if already registered)
	_ = api.RegisterPackage(server, regPayload, token)

	color.Cyan("Uploading container image payload to Aluminium server...\n")
	err = api.UploadPrebuilt(server, spec.Name, version, spec.Name, tarPath, token)
	if err != nil {
		return fmt.Errorf("failed to upload container image to Aluminium server: %w", err)
	}

	color.Green("✓ Successfully published container '%s@%s' to Aluminium server.\n", spec.Name, version)
	return nil
}
