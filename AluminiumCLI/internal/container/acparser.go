package container

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ParseACFile reads and parses a .ac container setup file into a ContainerSpec.
func ParseACFile(filePath string) (*ContainerSpec, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open container setup file: %w", err)
	}
	defer file.Close()

	return ParseACReader(file)
}

// ParseACReader reads from an io.Reader and parses the DSL into a ContainerSpec.
func ParseACReader(r io.Reader) (*ContainerSpec, error) {
	spec := NewContainerSpec()
	scanner := bufio.NewScanner(r)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		switch {
		case strings.HasPrefix(line, "use source"):
			parts := strings.Fields(line)
			if len(parts) < 3 {
				return nil, fmt.Errorf("line %d: invalid 'use source' directive: expected 'use source <source>'", lineNumber)
			}
			spec.Source = parts[2]

		case strings.HasPrefix(line, "use image"):
			parts := strings.Fields(line)
			if len(parts) < 3 {
				return nil, fmt.Errorf("line %d: invalid 'use image' directive: expected 'use image <image>'", lineNumber)
			}
			spec.Image = parts[2]

		case strings.HasPrefix(line, "use aluminium-packages:"):
			idx := strings.Index(line, ":")
			rawPkgs := strings.TrimSpace(line[idx+1:])
			if rawPkgs != "" {
				pkgs := strings.Fields(rawPkgs)
				spec.AluminiumPackages = append(spec.AluminiumPackages, pkgs...)
			}

		case strings.HasPrefix(line, "use aluminium-packages"):
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				spec.AluminiumPackages = append(spec.AluminiumPackages, parts[2:]...)
			}

		case strings.HasPrefix(line, "copy"):
			parts := strings.Fields(line)
			if len(parts) < 3 {
				return nil, fmt.Errorf("line %d: invalid 'copy' directive: expected 'copy <src> <dest>'", lineNumber)
			}
			spec.Steps = append(spec.Steps, BuildStep{
				Type:     "copy",
				CopySrc:  parts[1],
				CopyDest: parts[2],
			})

		case strings.HasPrefix(line, "wait package-installed"):
			parts := strings.Fields(line)
			if len(parts) < 3 {
				return nil, fmt.Errorf("line %d: invalid 'wait package-installed' directive: expected 'wait package-installed <pkg>'", lineNumber)
			}
			spec.Steps = append(spec.Steps, BuildStep{
				Type:        "wait",
				WaitType:    "package-installed",
				WaitPackage: parts[2],
			})

		case line == "wait all-packages-installed":
			spec.Steps = append(spec.Steps, BuildStep{
				Type:     "wait",
				WaitType: "all-packages-installed",
			})

		case strings.HasPrefix(line, "run"):
			cmdStr := strings.TrimSpace(line[3:])
			if cmdStr == "" {
				return nil, fmt.Errorf("line %d: invalid 'run' directive: expected command to run", lineNumber)
			}
			spec.Steps = append(spec.Steps, BuildStep{
				Type:       "run",
				RunCommand: cmdStr,
			})

		case strings.HasPrefix(line, "set interactive-shell-path"):
			parts := strings.Fields(line)
			if len(parts) < 3 {
				return nil, fmt.Errorf("line %d: invalid 'set interactive-shell-path' directive", lineNumber)
			}
			spec.InteractiveShellPath = parts[2]

		case strings.HasPrefix(line, "set interactive-shell"):
			parts := strings.Fields(line)
			if len(parts) < 3 {
				return nil, fmt.Errorf("line %d: invalid 'set interactive-shell' directive", lineNumber)
			}
			spec.InteractiveShell = (parts[2] == "true" || parts[2] == "1")

		case strings.HasPrefix(line, "set restart-policy"):
			parts := strings.Fields(line)
			if len(parts) < 3 {
				return nil, fmt.Errorf("line %d: invalid 'set restart-policy' directive", lineNumber)
			}
			spec.RestartPolicy = parts[2]

		case strings.HasPrefix(line, "set name"):
			parts := strings.Fields(line)
			if len(parts) < 3 {
				return nil, fmt.Errorf("line %d: invalid 'set name' directive", lineNumber)
			}
			spec.Name = parts[2]

		case strings.HasPrefix(line, "set replicas"):
			parts := strings.Fields(line)
			if len(parts) < 3 {
				return nil, fmt.Errorf("line %d: invalid 'set replicas' directive", lineNumber)
			}
			n, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid replicas number '%s': %w", lineNumber, parts[2], err)
			}
			spec.Replicas = n

		case line == "finalize":
			spec.Finalized = true

		default:
			return nil, fmt.Errorf("line %d: unknown syntax '%s'", lineNumber, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading container setup document: %w", err)
	}

	return spec, nil
}
