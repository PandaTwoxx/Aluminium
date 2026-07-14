package prompt

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

var buildSystemOptions = []huh.Option[string]{
	huh.NewOption("None (prebuilt only)", "none"),
	huh.NewOption("CMake", "cmake"),
	huh.NewOption("Make", "make"),
	huh.NewOption("Meson", "meson"),
	huh.NewOption("Custom scripts", "custom"),
}

// PackageRegisterInput holds values collected from the interactive register form.
type PackageRegisterInput struct {
	Name                  string
	Version               string
	BuildSystem           string
	SourceDir             string
	BuildFlags            string
	Dependencies          []string
	CustomBuildScript     string
	CustomInstallScript   string
	CustomUninstallScript string
}

// PackageRegister runs an interactive form for `aluminium package register`.
// Existing flag values are used as defaults where provided.
func PackageRegister(defaults PackageRegisterInput) (*PackageRegisterInput, error) {
	result := defaults
	if result.BuildSystem == "" {
		result.BuildSystem = "none"
	}

	depsInput := strings.Join(defaults.Dependencies, ", ")

	basicGroup := huh.NewGroup(
		huh.NewInput().
			Title("Package name").
			Description("Unique identifier for this package").
			Placeholder("mylib").
			Value(&result.Name).
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return fmt.Errorf("name is required")
				}
				return nil
			}),
		huh.NewInput().
			Title("Version").
			Description("Semantic version or release tag").
			Placeholder("1.0.0").
			Value(&result.Version).
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return fmt.Errorf("version is required")
				}
				return nil
			}),
		huh.NewSelect[string]().
			Title("Build system").
			Description("How source builds are performed on install").
			Options(buildSystemOptions...).
			Value(&result.BuildSystem),
		huh.NewInput().
			Title("Source directory or URL").
			Description("Cloneable URL or local path to package sources (optional for prebuilt-only packages)").
			Placeholder("https://github.com/org/mylib.git").
			Value(&result.SourceDir),
	).Title("Register Package").
		Description("Configure a new package on the Aluminium server")

	buildGroup := huh.NewGroup(
		huh.NewInput().
			Title("Build flags").
			Description("Arguments passed to cmake/make/meson (optional)").
			Placeholder("-DCMAKE_BUILD_TYPE=Release").
			Value(&result.BuildFlags),
		huh.NewInput().
			Title("Dependencies").
			Description("Comma-separated package specs, e.g. zlib@1.2.11, openssl@3.0").
			Placeholder("zlib@1.2.11").
			Value(&depsInput),
	)

	customGroup := huh.NewGroup(
		huh.NewText().
			Title("Custom build script").
			Description("Shell script run to compile the package").
			Value(&result.CustomBuildScript),
		huh.NewText().
			Title("Custom install script").
			Description("Shell script run to install built artifacts").
			Value(&result.CustomInstallScript),
		huh.NewText().
			Title("Custom uninstall script").
			Description("Shell script run when uninstalling (optional)").
			Value(&result.CustomUninstallScript),
	).WithHideFunc(func() bool {
		return result.BuildSystem != "custom"
	})

	form := huh.NewForm(basicGroup, buildGroup, customGroup)

	if err := form.Run(); err != nil {
		return nil, err
	}

	depsInput = strings.TrimSpace(depsInput)
	if depsInput != "" {
		parts := strings.Split(depsInput, ",")
		result.Dependencies = make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				result.Dependencies = append(result.Dependencies, trimmed)
			}
		}
	} else {
		result.Dependencies = nil
	}

	fmt.Println()
	fmt.Printf("  Name:         %s\n", result.Name)
	fmt.Printf("  Version:      %s\n", result.Version)
	fmt.Printf("  Build system: %s\n", result.BuildSystem)
	if result.SourceDir != "" {
		fmt.Printf("  Source:       %s\n", result.SourceDir)
	}
	if len(result.Dependencies) > 0 {
		fmt.Printf("  Dependencies: %s\n", strings.Join(result.Dependencies, ", "))
	}
	fmt.Println()

	confirmed, err := Confirm("Register this package?", true)
	if err != nil {
		return nil, err
	}
	if !confirmed {
		return nil, fmt.Errorf("registration cancelled")
	}

	return &result, nil
}
