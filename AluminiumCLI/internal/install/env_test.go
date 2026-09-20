package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// SanitizeHostPaths
// ---------------------------------------------------------------------------

func TestSanitizeHostPaths_stripsMatchingPrefixes(t *testing.T) {
	env := []string{
		"HOME=/home/user",
		"PATH=/opt/homebrew/bin:/usr/bin:/usr/local/bin",
		"LD_LIBRARY_PATH=/opt/homebrew/lib:/usr/lib",
		"PKG_CONFIG_PATH=/opt/homebrew/lib/pkgconfig",
		"TERM=xterm-256color",
		"DYLD_LIBRARY_PATH=/opt/homebrew/lib:/home/linuxbrew/.linuxbrew/lib",
	}
	strip := []string{"/opt/homebrew", "/home/linuxbrew"}

	got := SanitizeHostPaths(env, strip)

	pathVal := envVal(got, "PATH")
	if strings.Contains(pathVal, "/opt/homebrew") {
		t.Errorf("PATH still contains /opt/homebrew: %s", pathVal)
	}
	if !strings.Contains(pathVal, "/usr/bin") {
		t.Errorf("PATH should contain /usr/bin: %s", pathVal)
	}

	ldVal := envVal(got, "LD_LIBRARY_PATH")
	if strings.Contains(ldVal, "/opt/homebrew") {
		t.Errorf("LD_LIBRARY_PATH still contains /opt/homebrew: %s", ldVal)
	}
	if !strings.Contains(ldVal, "/usr/lib") {
		t.Errorf("LD_LIBRARY_PATH should contain /usr/lib: %s", ldVal)
	}

	// Non-path var must pass through unchanged.
	if envVal(got, "HOME") != "/home/user" {
		t.Errorf("HOME should not be modified")
	}
	if envVal(got, "TERM") != "xterm-256color" {
		t.Errorf("TERM should not be modified")
	}

	// PKG_CONFIG_PATH was entirely within the stripped prefix — key must be
	// absent (empty value dropped) or value must be empty.
	pkgVal := envVal(got, "PKG_CONFIG_PATH")
	if strings.Contains(pkgVal, "/opt/homebrew") {
		t.Errorf("PKG_CONFIG_PATH still contains /opt/homebrew: %s", pkgVal)
	}

	// DYLD_LIBRARY_PATH had both homebrew and linuxbrew — both stripped.
	dyldVal := envVal(got, "DYLD_LIBRARY_PATH")
	if strings.Contains(dyldVal, "/opt/homebrew") || strings.Contains(dyldVal, "/home/linuxbrew") {
		t.Errorf("DYLD_LIBRARY_PATH still contains stripped prefix: %s", dyldVal)
	}
}

func TestSanitizeHostPaths_noOp_whenNoPrefixMatches(t *testing.T) {
	env := []string{
		"PATH=/usr/bin:/usr/local/bin",
		"HOME=/home/user",
	}
	got := SanitizeHostPaths(env, []string{"/opt/homebrew"})
	if envVal(got, "PATH") != "/usr/bin:/usr/local/bin" {
		t.Errorf("PATH should be unchanged: %s", envVal(got, "PATH"))
	}
}

// ---------------------------------------------------------------------------
// filterPathComponents
// ---------------------------------------------------------------------------

func TestFilterPathComponents_removesMatchingPrefix(t *testing.T) {
	result := filterPathComponents("/opt/homebrew/bin:/usr/bin:/usr/local/bin", []string{"/opt/homebrew"})
	if strings.Contains(result, "/opt/homebrew") {
		t.Errorf("result should not contain /opt/homebrew: %s", result)
	}
	if !strings.Contains(result, "/usr/bin") {
		t.Errorf("result should contain /usr/bin: %s", result)
	}
}

func TestFilterPathComponents_emptyInput(t *testing.T) {
	result := filterPathComponents("", []string{"/opt/homebrew"})
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestFilterPathComponents_noMatchLeavesFull(t *testing.T) {
	input := "/usr/bin:/usr/local/bin"
	result := filterPathComponents(input, []string{"/opt/homebrew"})
	if result != input {
		t.Errorf("expected %q unchanged, got %q", input, result)
	}
}

// ---------------------------------------------------------------------------
// findSSLCertBundle
// ---------------------------------------------------------------------------

func TestFindSSLCertBundle_aluminium_openssl_preferred(t *testing.T) {
	base := t.TempDir()

	// Create a mock Aluminium-installed OpenSSL cert bundle.
	certDir := filepath.Join(base, "openssl", "etc", "ssl")
	if err := os.MkdirAll(certDir, 0755); err != nil {
		t.Fatal(err)
	}
	certFile := filepath.Join(certDir, "cert.pem")
	if err := os.WriteFile(certFile, []byte("# mock cert"), 0644); err != nil {
		t.Fatal(err)
	}

	gotFile, _ := findSSLCertBundle(base)
	if gotFile != certFile {
		t.Errorf("expected Aluminium OpenSSL cert %q, got %q", certFile, gotFile)
	}
}

func TestFindSSLCertBundle_returnsEmpty_whenNoneExist(t *testing.T) {
	// Use a temp dir guaranteed to have no cert files, and pass it as
	// installBase so only the Aluminium-internal paths are checked
	// (system paths such as /etc/ssl may exist on the host — we don't
	// assert on those here).
	base := t.TempDir()
	gotFile, _ := findSSLCertBundle(base)
	// If the host has no /etc/ssl/cert.pem either, both should be empty.
	// We only assert that the Aluminium-specific path was not returned.
	if gotFile == filepath.Join(base, "openssl", "etc", "ssl", "cert.pem") {
		t.Errorf("should not return non-existent cert path")
	}
}

func TestFindSSLCertBundle_certDir_detected(t *testing.T) {
	base := t.TempDir()

	// Create a mock Aluminium-installed cert directory.
	certDirPath := filepath.Join(base, "openssl", "etc", "ssl", "certs")
	if err := os.MkdirAll(certDirPath, 0755); err != nil {
		t.Fatal(err)
	}

	_, gotDir := findSSLCertBundle(base)
	if gotDir != certDirPath {
		t.Errorf("expected cert dir %q, got %q", certDirPath, gotDir)
	}
}

// ---------------------------------------------------------------------------
// buildEnv
// ---------------------------------------------------------------------------

func TestBuildEnv_noHomebrewPaths(t *testing.T) {
	// Temporarily inject a Homebrew path into PATH so we can confirm it is
	// stripped by buildEnv.
	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", "/opt/homebrew/bin:"+origPath)
	defer os.Setenv("PATH", origPath)

	configDir := t.TempDir()
	state := &InstalledState{Packages: make(map[string]InstalledPackage)}

	env := buildEnv(configDir, state)

	pathVal := envVal(env, "PATH")
	if strings.Contains(pathVal, "/opt/homebrew") {
		t.Errorf("buildEnv PATH still contains /opt/homebrew: %s", pathVal)
	}
}

func TestBuildEnv_containsAluminiumBins_whenPackageInstalled(t *testing.T) {
	configDir := t.TempDir()

	// Simulate an installed package with a bin/ directory.
	pkgBin := filepath.Join(configDir, "install", "mypkg", "bin")
	if err := os.MkdirAll(pkgBin, 0755); err != nil {
		t.Fatal(err)
	}

	state := &InstalledState{
		Packages: map[string]InstalledPackage{
			"mypkg": {Version: "1.0", Server: "https://example.com"},
		},
	}

	env := buildEnv(configDir, state)

	pathVal := envVal(env, "PATH")
	if !strings.Contains(pathVal, pkgBin) {
		t.Errorf("buildEnv PATH should contain %q, got %q", pkgBin, pathVal)
	}
}

func TestBuildEnv_doesNotInheritHostLibraryPaths(t *testing.T) {
	origLD := os.Getenv("LD_LIBRARY_PATH")
	_ = os.Setenv("LD_LIBRARY_PATH", "/opt/homebrew/lib:/some/other/lib")
	defer func() { _ = os.Setenv("LD_LIBRARY_PATH", origLD) }()

	configDir := t.TempDir()
	state := &InstalledState{Packages: make(map[string]InstalledPackage)}

	env := buildEnv(configDir, state)

	// LD_LIBRARY_PATH from the host must never appear in the clean env.
	for _, kv := range env {
		if strings.HasPrefix(kv, "LD_LIBRARY_PATH=") {
			if strings.Contains(kv, "/opt/homebrew") || strings.Contains(kv, "/some/other/lib") {
				t.Errorf("buildEnv leaked host LD_LIBRARY_PATH: %s", kv)
			}
		}
	}
}

func TestBuildEnv_sslCertInjected_whenBundleExists(t *testing.T) {
	configDir := t.TempDir()

	// Create a mock cert bundle inside the Aluminium install dir.
	certPath := filepath.Join(configDir, "install", "openssl", "etc", "ssl", "cert.pem")
	if err := os.MkdirAll(filepath.Dir(certPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(certPath, []byte("# mock cert"), 0644); err != nil {
		t.Fatal(err)
	}

	state := &InstalledState{Packages: make(map[string]InstalledPackage)}
	env := buildEnv(configDir, state)

	if envVal(env, "SSL_CERT_FILE") != certPath {
		t.Errorf("expected SSL_CERT_FILE=%q, got %q", certPath, envVal(env, "SSL_CERT_FILE"))
	}
	if envVal(env, "REQUESTS_CA_BUNDLE") != certPath {
		t.Errorf("expected REQUESTS_CA_BUNDLE=%q, got %q", certPath, envVal(env, "REQUESTS_CA_BUNDLE"))
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// envVal extracts the value for the given key from an "KEY=VALUE" env slice.
// Returns "" if the key is not found.
func envVal(env []string, key string) string {
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			return kv[len(prefix):]
		}
	}
	return ""
}
