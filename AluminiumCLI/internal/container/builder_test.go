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
		AluminiumPackages: []string{"libfoo@1.0", "libbar@2.0"},
		Steps: []BuildStep{
			{Type: "copy", CopySrc: "local.sh", CopyDest: "container.sh"},
			{Type: "wait", WaitType: "package-installed", WaitPackage: "libfoo@1.0"},
			{Type: "run", RunCommand: "container.sh"},
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
	if !strings.Contains(content, "COPY local.sh container.sh") {
		t.Errorf("expected COPY step in Dockerfile")
	}
	if !strings.Contains(content, "apt-get install -y libfoo") {
		t.Errorf("expected libfoo package install in Dockerfile")
	}
	if !strings.Contains(content, "apt-get install -y libbar") {
		t.Errorf("expected libbar package install in Dockerfile")
	}
	if !strings.Contains(content, "RUN echo hello") {
		t.Errorf("expected RUN echo hello in Dockerfile")
	}
}
