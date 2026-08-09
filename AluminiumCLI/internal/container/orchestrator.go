package container

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// LaunchContainer starts container instances according to ContainerSpec and image tag.
func LaunchContainer(spec *ContainerSpec, imageTag string) error {
	if err := EnsureRuntimeRunning(); err != nil {
		return err
	}

	nerdctlBin := GetToolPath("nerdctl")
	baseName := spec.Name
	if baseName == "" {
		baseName = "aluminium-app"
	}

	replicas := spec.Replicas
	if replicas < 1 {
		replicas = 1
	}

	restartPolicy := spec.RestartPolicy
	if restartPolicy == "" {
		restartPolicy = "no"
	}

	// Interactive single container launch
	if spec.InteractiveShell && replicas == 1 {
		shellPath := spec.InteractiveShellPath
		if shellPath == "" {
			shellPath = "/bin/sh"
		}

		// Remove existing container with same name if any
		_ = exec.Command(nerdctlBin, "rm", "-f", baseName).Run()

		fmt.Printf("Launching container '%s' interactively with shell '%s'...\n", baseName, shellPath)

		args := []string{"run", "-it", "--name", baseName}
		if restartPolicy != "no" {
			args = append(args, fmt.Sprintf("--restart=%s", restartPolicy))
		}
		args = append(args, imageTag, shellPath)

		cmd := exec.Command(nerdctlBin, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd.Run()
	}

	// Scaled or background container launch
	fmt.Printf("Launching %d container replica(s) for '%s' (image: %s)...\n", replicas, baseName, imageTag)

	for i := 1; i <= replicas; i++ {
		var containerName string
		if replicas == 1 {
			containerName = baseName
		} else {
			containerName = fmt.Sprintf("%s-%d", baseName, i)
		}

		// Cleanup existing container with same name if any
		_ = exec.Command(nerdctlBin, "rm", "-f", containerName).Run()

		args := []string{"run", "-d", "--name", containerName}
		if restartPolicy != "no" {
			args = append(args, fmt.Sprintf("--restart=%s", restartPolicy))
		}
		args = append(args, imageTag)

		cmd := exec.Command(nerdctlBin, args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to launch container replica '%s': %w\nOutput: %s", containerName, err, string(output))
		}

		containerID := strings.TrimSpace(string(output))
		if len(containerID) > 12 {
			containerID = containerID[:12]
		}
		fmt.Printf("  ✓ Started container '%s' (ID: %s) [restart: %s]\n", containerName, containerID, restartPolicy)
	}

	fmt.Printf("Successfully launched %d replica(s).\n", replicas)
	return nil
}

// ListContainerReplicas returns running container replica names for a given base name.
func ListContainerReplicas(baseName string) ([]string, error) {
	nerdctlBin := GetToolPath("nerdctl")
	cmd := exec.Command(nerdctlBin, "ps", "--format", "{{.Names}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var replicas []string
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == baseName || strings.HasPrefix(name, baseName+"-") {
			replicas = append(replicas, name)
		}
	}
	return replicas, nil
}

// FormatReplicasInfo returns string representation of container replicas count.
func FormatReplicasInfo(count int) string {
	return strconv.Itoa(count)
}
