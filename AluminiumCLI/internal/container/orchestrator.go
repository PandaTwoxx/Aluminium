package container

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
)

type ContainerInfo struct {
	ID        string `json:"ID"`
	Names     string `json:"Names"`
	Image     string `json:"Image"`
	Status    string `json:"Status"`
	CreatedAt string `json:"CreatedAt"`
	Command   string `json:"Command"`
}

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

		color.Cyan("Launching container '%s' interactively with shell '%s'...\n", baseName, shellPath)

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
	color.Cyan("Launching %d container replica(s) for '%s' (image: %s)...\n", replicas, baseName, imageTag)

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
		color.Green("  ✓ Started container '%s' (ID: %s) [restart: %s]\n", containerName, containerID, restartPolicy)
	}

	color.Green("Successfully launched %d replica(s).\n", replicas)
	return nil
}

// ListContainers returns containers running or all containers if showAll is true.
func ListContainers(showAll bool) ([]ContainerInfo, error) {
	if err := EnsureRuntimeRunning(); err != nil {
		return nil, err
	}

	nerdctlBin := GetToolPath("nerdctl")
	args := []string{"ps", "--format", "{{json .}}"}
	if showAll {
		args = append(args, "-a")
	}

	cmd := exec.Command(nerdctlBin, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var containers []ContainerInfo
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var c ContainerInfo
		if err := json.Unmarshal([]byte(line), &c); err == nil {
			containers = append(containers, c)
		} else {
			// Fallback plain parsing if json unmarshal fails
			parts := strings.Split(line, "\t")
			if len(parts) >= 2 {
				containers = append(containers, ContainerInfo{
					ID:    parts[0],
					Names: parts[1],
				})
			}
		}
	}

	return containers, nil
}

// PrintContainersTable outputs a styled table of containers using color headers.
func PrintContainersTable(containers []ContainerInfo) {
	if len(containers) == 0 {
		color.Yellow("No containers found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	color.Cyan("CONTAINER ID\tNAMES\tIMAGE\tSTATUS\tCREATED")
	fmt.Fprintln(w, "ID\tNAMES\tIMAGE\tSTATUS\tCREATED")

	for _, c := range containers {
		id := c.ID
		if len(id) > 12 {
			id = id[:12]
		}
		statusColor := color.YellowString(c.Status)
		if strings.Contains(strings.ToLower(c.Status), "up") || strings.Contains(strings.ToLower(c.Status), "running") {
			statusColor = color.GreenString(c.Status)
		} else if strings.Contains(strings.ToLower(c.Status), "exited") {
			statusColor = color.RedString(c.Status)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			color.CyanString(id),
			color.WhiteString(c.Names),
			c.Image,
			statusColor,
			c.CreatedAt,
		)
	}
	w.Flush()
}

// StartContainer starts a container by name or ID.
func StartContainer(nameOrID string) error {
	if err := EnsureRuntimeRunning(); err != nil {
		return err
	}

	nerdctlBin := GetToolPath("nerdctl")
	color.Cyan("Starting container '%s'...\n", nameOrID)

	cmd := exec.Command(nerdctlBin, "start", nameOrID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start container '%s': %w\nOutput: %s", nameOrID, err, string(out))
	}

	color.Green("✓ Started container '%s'.\n", nameOrID)
	return nil
}

// StopContainer stops a running container by name or ID.
func StopContainer(nameOrID string) error {
	if err := EnsureRuntimeRunning(); err != nil {
		return err
	}

	nerdctlBin := GetToolPath("nerdctl")
	color.Cyan("Stopping container '%s'...\n", nameOrID)

	cmd := exec.Command(nerdctlBin, "stop", nameOrID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop container '%s': %w\nOutput: %s", nameOrID, err, string(out))
	}

	color.Green("✓ Stopped container '%s'.\n", nameOrID)
	return nil
}

// RestartContainer restarts a container by name or ID.
func RestartContainer(nameOrID string) error {
	if err := EnsureRuntimeRunning(); err != nil {
		return err
	}

	nerdctlBin := GetToolPath("nerdctl")
	color.Cyan("Restarting container '%s'...\n", nameOrID)

	cmd := exec.Command(nerdctlBin, "restart", nameOrID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart container '%s': %w\nOutput: %s", nameOrID, err, string(out))
	}

	color.Green("✓ Restarted container '%s'.\n", nameOrID)
	return nil
}

// DeleteContainer removes a container by name or ID.
func DeleteContainer(nameOrID string, force bool) error {
	if err := EnsureRuntimeRunning(); err != nil {
		return err
	}

	nerdctlBin := GetToolPath("nerdctl")
	color.Cyan("Deleting container '%s'...\n", nameOrID)

	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, nameOrID)

	cmd := exec.Command(nerdctlBin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete container '%s': %w\nOutput: %s", nameOrID, err, string(out))
	}

	color.Green("✓ Deleted container '%s'.\n", nameOrID)
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
