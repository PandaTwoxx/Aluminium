package prompt

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
)

type ContainerLaunchInput struct {
	Target        string
	Source        string
	Name          string
	Replicas      int
	RestartPolicy string
	Interactive   bool
}

type ContainerOption struct {
	ID     string
	Name   string
	Status string
}

var sourceOptions = []huh.Option[string]{
	huh.NewOption("Docker Hub", "dockerhub"),
	huh.NewOption("Aluminium Registry", "aluminium"),
}

var restartOptions = []huh.Option[string]{
	huh.NewOption("No (never restart)", "no"),
	huh.NewOption("Always restart", "always"),
	huh.NewOption("Unless stopped", "unless-stopped"),
	huh.NewOption("On failure", "on-failure"),
}

// ContainerLaunchPrompt displays an interactive form for launching a container environment.
func ContainerLaunchPrompt(defaults ContainerLaunchInput) (*ContainerLaunchInput, error) {
	result := defaults
	if result.Source == "" {
		result.Source = "dockerhub"
	}
	if result.RestartPolicy == "" {
		result.RestartPolicy = "no"
	}
	replicasStr := strconv.Itoa(result.Replicas)
	if result.Replicas <= 0 {
		replicasStr = "1"
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Target setup document (.ac) or image name").
				Description("e.g., container-setup.ac or ubuntu:latest").
				Value(&result.Target).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("target setup file or image name is required")
					}
					return nil
				}),
			huh.NewSelect[string]().
				Title("Source registry").
				Options(sourceOptions...).
				Value(&result.Source),
			huh.NewInput().
				Title("Container base name").
				Description("Unique name for the container group (optional)").
				Value(&result.Name),
			huh.NewInput().
				Title("Number of replicas").
				Description("Scale container across N instances").
				Value(&replicasStr).
				Validate(func(s string) error {
					n, err := strconv.Atoi(strings.TrimSpace(s))
					if err != nil || n < 1 {
						return fmt.Errorf("replicas must be a positive integer")
					}
					return nil
				}),
			huh.NewSelect[string]().
				Title("Restart policy").
				Options(restartOptions...).
				Value(&result.RestartPolicy),
			huh.NewConfirm().
				Title("Interactive shell mode?").
				Description("Attach interactive terminal shell to container").
				Value(&result.Interactive),
		).Title("Launch Container Environment").
			Description("Configure scalable container setup"),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	n, _ := strconv.Atoi(strings.TrimSpace(replicasStr))
	result.Replicas = n
	return &result, nil
}

// ContainerSelectPrompt presents a selectable list of containers for lifecycle commands (stop, start, restart, delete).
func ContainerSelectPrompt(containers []ContainerOption, title string) (string, error) {
	if len(containers) == 0 {
		var customInput string
		err := huh.NewInput().
			Title(title).
			Description("Enter container name or ID").
			Value(&customInput).
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return fmt.Errorf("container name or ID is required")
				}
				return nil
			}).
			Run()
		return strings.TrimSpace(customInput), err
	}

	options := make([]huh.Option[string], 0, len(containers))
	for _, c := range containers {
		id := c.ID
		if len(id) > 8 {
			id = id[:8]
		}
		label := fmt.Sprintf("%s (%s) [%s]", c.Name, id, c.Status)
		value := c.Name
		if value == "" {
			value = c.ID
		}
		options = append(options, huh.NewOption(label, value))
	}

	var selected string
	err := huh.NewSelect[string]().
		Title(title).
		Description("Select target container").
		Options(options...).
		Value(&selected).
		Run()

	if err != nil {
		return "", err
	}
	return selected, nil
}
