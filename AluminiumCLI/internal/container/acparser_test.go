package container

import (
	"strings"
	"testing"
)

func TestParseACReader_Example(t *testing.T) {
	input := `use source dockerhub
use image ubuntu:latest
use aluminium-packages: python3@3.14.3 node@25

copy local-python-file.py container-file.py
wait package-installed python3@3.14.3
run container-file.py

wait all-packages-installed
run echo "this is a test script"
set interactive-shell true
set interactive-shell-path /bin/sh
set restart-policy always
set name test
set replicas 4
finalize`

	spec, err := ParseACReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error parsing AC spec: %v", err)
	}

	if spec.Source != "dockerhub" {
		t.Errorf("expected Source dockerhub, got %s", spec.Source)
	}
	if spec.Image != "ubuntu:latest" {
		t.Errorf("expected Image ubuntu:latest, got %s", spec.Image)
	}
	if len(spec.AluminiumPackages) != 2 || spec.AluminiumPackages[0] != "python3@3.14.3" || spec.AluminiumPackages[1] != "node@25" {
		t.Errorf("expected packages [python3@3.14.3 node@25], got %v", spec.AluminiumPackages)
	}
	if !spec.InteractiveShell {
		t.Errorf("expected InteractiveShell true")
	}
	if spec.InteractiveShellPath != "/bin/sh" {
		t.Errorf("expected InteractiveShellPath /bin/sh, got %s", spec.InteractiveShellPath)
	}
	if spec.RestartPolicy != "always" {
		t.Errorf("expected RestartPolicy always, got %s", spec.RestartPolicy)
	}
	if spec.Name != "test" {
		t.Errorf("expected Name test, got %s", spec.Name)
	}
	if spec.Replicas != 4 {
		t.Errorf("expected Replicas 4, got %d", spec.Replicas)
	}
	if !spec.Finalized {
		t.Errorf("expected Finalized true")
	}

	if len(spec.Steps) != 5 {
		t.Fatalf("expected 5 steps, got %d", len(spec.Steps))
	}
	if spec.Steps[0].Type != "copy" || spec.Steps[0].CopySrc != "local-python-file.py" || spec.Steps[0].CopyDest != "container-file.py" {
		t.Errorf("step 0 mismatch: %+v", spec.Steps[0])
	}
	if spec.Steps[1].Type != "wait" || spec.Steps[1].WaitType != "package-installed" || spec.Steps[1].WaitPackage != "python3@3.14.3" {
		t.Errorf("step 1 mismatch: %+v", spec.Steps[1])
	}
	if spec.Steps[2].Type != "run" || spec.Steps[2].RunCommand != "container-file.py" {
		t.Errorf("step 2 mismatch: %+v", spec.Steps[2])
	}
	if spec.Steps[3].Type != "wait" || spec.Steps[3].WaitType != "all-packages-installed" {
		t.Errorf("step 3 mismatch: %+v", spec.Steps[3])
	}
	if spec.Steps[4].Type != "run" || spec.Steps[4].RunCommand != "echo \"this is a test script\"" {
		t.Errorf("step 4 mismatch: %+v", spec.Steps[4])
	}
}
