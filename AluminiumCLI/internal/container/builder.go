package container

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BuildContainerImage constructs a Dockerfile context from ContainerSpec and builds it using nerdctl.
func BuildContainerImage(spec *ContainerSpec, workDir string) (string, error) {
	if spec.Name == "" {
		spec.Name = "aluminium-container"
	}
	if spec.Image == "" {
		spec.Image = "ubuntu:latest"
	}

	// Create temporary build directory if workDir not specified
	buildCtxDir, err := os.MkdirTemp("", "aluminium-build-*")
	if err != nil {
		return "", fmt.Errorf("failed to create build context directory: %w", err)
	}
	defer os.RemoveAll(buildCtxDir)

	dockerfileContent, err := GenerateDockerfile(spec, workDir, buildCtxDir)
	if err != nil {
		return "", fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	dockerfilePath := filepath.Join(buildCtxDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	tag := fmt.Sprintf("%s:latest", spec.Name)
	nerdctlBin := GetToolPath("nerdctl")

	fmt.Printf("Building container image '%s' using nerdctl...\n", tag)
	cmd := exec.Command(nerdctlBin, "build", "-t", tag, buildCtxDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("nerdctl build failed: %w", err)
	}

	fmt.Printf("Successfully built container image '%s'.\n", tag)
	return tag, nil
}

// GenerateDockerfile creates Dockerfile contents and prepares build context files.
func GenerateDockerfile(spec *ContainerSpec, srcBaseDir string, buildCtxDir string) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("FROM %s\n\n", spec.Image))
	sb.WriteString("ENV DEBIAN_FRONTEND=noninteractive\n")
	sb.WriteString("ENV PATH=\"/root/.aluminium/install/bin:/usr/local/bin:${PATH}\"\n")
	sb.WriteString("WORKDIR /app\n\n")

	// Helper layer to ensure basic tools exist (curl, ca-certificates, bash)
	sb.WriteString("# Base dependencies installation\n")
	sb.WriteString("RUN if command -v apt-get >/dev/null 2>&1; then apt-get update && apt-get install -y curl ca-certificates bash build-essential || true; " +
		"elif command -v apk >/dev/null 2>&1; then apk add --no-cache curl ca-certificates bash build-base || true; " +
		"elif command -v dnf >/dev/null 2>&1; then dnf install -y curl ca-certificates bash || true; fi\n\n")

	// Embed host aluminium binary into container context if available
	hasAluminiumBin := false
	if execPath, err := os.Executable(); err == nil {
		dstBinPath := filepath.Join(buildCtxDir, "aluminium")
		if err := copyFile(execPath, dstBinPath); err == nil {
			hasAluminiumBin = true
		}
	} else if path, err := exec.LookPath("aluminium"); err == nil {
		dstBinPath := filepath.Join(buildCtxDir, "aluminium")
		if err := copyFile(path, dstBinPath); err == nil {
			hasAluminiumBin = true
		}
	}

	if hasAluminiumBin {
		sb.WriteString("# Embed Aluminium CLI into container\n")
		sb.WriteString("COPY aluminium /usr/local/bin/aluminium\n")
		sb.WriteString("RUN chmod +x /usr/local/bin/aluminium\n\n")
	}

	installedPackages := make(map[string]bool)

	installPackageLayer := func(pkgSpec string) {
		if installedPackages[pkgSpec] {
			return
		}
		installedPackages[pkgSpec] = true
		pkgName := pkgSpec
		if idx := strings.Index(pkgSpec, "@"); idx != -1 {
			pkgName = pkgSpec[:idx]
		}

		sb.WriteString(fmt.Sprintf("# Install package '%s' with Aluminium CLI inside container\n", pkgSpec))
		if hasAluminiumBin {
			sb.WriteString(fmt.Sprintf("RUN aluminium install --no-interactive %s || "+
				"(if command -v apt-get >/dev/null 2>&1; then apt-get update && apt-get install -y %s || true; "+
				"elif command -v apk >/dev/null 2>&1; then apk add %s || true; "+
				"elif command -v dnf >/dev/null 2>&1; then dnf install -y %s || true; fi)\n\n", pkgSpec, pkgName, pkgName, pkgName))
		} else {
			sb.WriteString(fmt.Sprintf("RUN if command -v apt-get >/dev/null 2>&1; then apt-get update && apt-get install -y %s || true; "+
				"elif command -v apk >/dev/null 2>&1; then apk add %s || true; "+
				"elif command -v dnf >/dev/null 2>&1; then dnf install -y %s || true; fi\n\n", pkgName, pkgName, pkgName))
		}
	}

	for _, step := range spec.Steps {
		switch step.Type {
		case "copy":
			// Copy local file into build context
			srcPath := step.CopySrc
			if !filepath.IsAbs(srcPath) && srcBaseDir != "" {
				srcPath = filepath.Join(srcBaseDir, step.CopySrc)
			}
			destCtxPath := filepath.Join(buildCtxDir, filepath.Base(step.CopySrc))

			if err := copyFileOrDir(srcPath, destCtxPath); err != nil {
				// If source file not found locally, create stub or warning
				fmt.Printf("Warning: file '%s' not found locally for COPY step, creating placeholder.\n", srcPath)
				_ = os.WriteFile(destCtxPath, []byte("#!/bin/sh\necho 'placeholder'\n"), 0755)
			}

			sb.WriteString(fmt.Sprintf("COPY %s %s\n", filepath.Base(step.CopySrc), step.CopyDest))
			sb.WriteString(fmt.Sprintf("RUN if [ -f \"%s\" ]; then chmod +x \"%s\" || true; fi\n\n", step.CopyDest, step.CopyDest))

		case "wait":
			if step.WaitType == "package-installed" && step.WaitPackage != "" {
				installPackageLayer(step.WaitPackage)
			} else if step.WaitType == "all-packages-installed" {
				for _, pkg := range spec.AluminiumPackages {
					installPackageLayer(pkg)
				}
			}

		case "run":
			// If running a script file copied into container, allow shell fallback execution
			cmd := step.RunCommand
			if !strings.Contains(cmd, " ") && !strings.HasPrefix(cmd, "/") && !strings.HasPrefix(cmd, "./") {
				sb.WriteString(fmt.Sprintf("RUN if [ -f \"%s\" ]; then ./%s; else %s; fi\n\n", cmd, cmd, cmd))
			} else {
				sb.WriteString(fmt.Sprintf("RUN %s\n\n", cmd))
			}
		}
	}

	// Install any remaining packages from use aluminium-packages:
	for _, pkg := range spec.AluminiumPackages {
		if !installedPackages[pkg] {
			installPackageLayer(pkg)
		}
	}

	return sb.String(), nil
}

func copyFileOrDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return os.Chmod(dst, 0755)
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if err := copyFileOrDir(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}
