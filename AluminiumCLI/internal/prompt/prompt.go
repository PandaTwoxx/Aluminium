package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
)

// SimpleConfirm prompts with a plain [y/N] line when huh is unavailable.
func SimpleConfirm(message string, defaultNo bool) (bool, error) {
	defaultHint := "y/N"
	if !defaultNo {
		defaultHint = "Y/n"
	}
	fmt.Printf("%s [%s]: ", message, defaultHint)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultNo, err
	}
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return !defaultNo, nil
	}
	return input == "y" || input == "yes", nil
}

// Confirm shows a styled yes/no prompt.
func Confirm(title string, defaultYes bool) (bool, error) {
	result := defaultYes
	err := huh.NewConfirm().
		Title(title).
		Affirmative("Yes").
		Negative("No").
		Value(&result).
		Run()
	if err != nil {
		return defaultYes, err
	}
	return result, nil
}

// PackageSpecs collects one or more package specs for an interactive install.
func PackageSpecs() ([]string, error) {
	var value string
	err := huh.NewInput().
		Title("Packages to install").
		Description("Comma-separated package specs, for example zlib@1.2.11, openssl").
		Value(&value).
		Validate(func(s string) error {
			if strings.TrimSpace(s) == "" {
				return fmt.Errorf("at least one package is required")
			}
			return nil
		}).
		Run()
	if err != nil {
		return nil, err
	}
	parts := strings.Split(value, ",")
	packages := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			packages = append(packages, trimmed)
		}
	}
	return packages, nil
}

func OutputDirectory(defaultValue string) (string, error) {
	value := defaultValue
	err := huh.NewInput().
		Title("Forge output directory").
		Description("Directory where the forge package will be installed").
		Value(&value).
		Validate(requiredValue("output directory")).
		Run()
	return value, err
}

type PackageOperationInput struct {
	Name    string
	Version string
	File    string
	Output  string
}

func PackageOperation(defaults PackageOperationInput, needFile, needOutput bool) (*PackageOperationInput, error) {
	result := defaults
	fields := []huh.Field{
		huh.NewInput().Title("Package name").Value(&result.Name).Validate(requiredValue("package name")),
		huh.NewInput().Title("Version").Value(&result.Version).Validate(requiredValue("version")),
	}
	if needFile {
		fields = append(fields, huh.NewInput().Title("Prebuilt file").Value(&result.File).Validate(requiredValue("prebuilt file")))
	}
	if needOutput {
		fields = append(fields, huh.NewInput().Title("Output file").Value(&result.Output).Validate(requiredValue("output file")))
	}
	if err := huh.NewForm(huh.NewGroup(fields...)).Title("Package operation").Run(); err != nil {
		return nil, err
	}
	return &result, nil
}

func requiredValue(label string) func(string) error {
	return func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", label)
		}
		return nil
	}
}
