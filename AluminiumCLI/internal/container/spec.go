package container

type BuildStep struct {
	Type        string // "copy", "wait", "run"
	CopySrc     string
	CopyDest    string
	WaitType    string // "package-installed", "all-packages-installed"
	WaitPackage string
	RunCommand  string
}

type ContainerSpec struct {
	Source               string      `json:"source"`
	Image                string      `json:"image"`
	AluminiumPackages    []string    `json:"aluminium_packages"`
	Steps                []BuildStep `json:"steps"`
	InteractiveShell     bool        `json:"interactive_shell"`
	InteractiveShellPath string      `json:"interactive_shell_path"`
	RestartPolicy        string      `json:"restart_policy"`
	Name                 string      `json:"name"`
	Replicas             int         `json:"replicas"`
	Finalized            bool        `json:"finalized"`
}

func NewContainerSpec() *ContainerSpec {
	return &ContainerSpec{
		Source:               "dockerhub",
		AluminiumPackages:    make([]string, 0),
		Steps:                make([]BuildStep, 0),
		InteractiveShellPath: "/bin/sh",
		RestartPolicy:        "no",
		Replicas:             1,
	}
}
