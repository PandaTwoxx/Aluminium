package container

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateDockerfile(t *testing.T) {
	spec := &ContainerSpec{
		Source:            "dockerhub",
		Image:             "ubuntu:latest",
		AluminiumPackages: []string{"python3@3.14.3", "node@25"},
		Steps: []BuildStep{
			{Type: "copy", CopySrc: "local.py", CopyDest: "container.py"},
			{Type: "wait", WaitType: "package-installed", WaitPackage: "python3@3.14.3"},
			{Type: "run", RunCommand: "container.py"},
			{Type: "wait", WaitType: "all-packages-installed"},
			{Type: "run", RunCommand: "echo hello"},
		},
		Name: "test",
	}

	tmpDir, err := os.MkdirTemp("", "test-dockerfile-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content, err := GenerateDockerfile(spec, "", tmpDir)
	if err != nil {
		t.Fatalf("GenerateDockerfile failed: %v", err)
	}

	if !strings.Contains(content, "FROM ubuntu:latest") {
		t.Errorf("expected FROM ubuntu:latest in Dockerfile")
	}
	if !strings.Contains(content, "COPY local.py container.py") {
		t.Errorf("expected COPY step in Dockerfile")
	}
	if !strings.Contains(content, "apt-get install -y python3") {
		t.Errorf("expected python3 package install in Dockerfile")
	}
	if !strings.Contains(content, "apt-get install -y node") {
		t.Errorf("expected node package install in Dockerfile")
	}
	if !strings.Contains(content, "RUN echo hello") {
		t.Errorf("expected RUN echo hello in Dockerfile")
	}
}
