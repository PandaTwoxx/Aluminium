package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Package struct {
	ID               string      `json:"_id,omitempty"`
	Name             string      `json:"name"`
	Version          string      `json:"version"`
	BuildSystem      string      `json:"buildSystem"`
	Dependencies     []string    `json:"dependencies"`
	PrebuiltBinaries []string    `json:"prebuiltBinaries"`
	BuildSetup       *BuildSetup `json:"buildSetup,omitempty"`
	Owner            string      `json:"owner,omitempty"`
	UploadedAt       string      `json:"uploadedAt,omitempty"`
}

type BuildSetup struct {
	BuildScript     string `json:"buildScript"`
	InstallScript   string `json:"installScript"`
	UninstallScript string `json:"uninstallScript"`
	SourceCodeURL   string `json:"sourceCodeUrl,omitempty"`
}

type TokenInfo struct {
	First8Chars string   `json:"first8Chars"`
	CreatedAt   string   `json:"createdAt"`
	Scopes      []string `json:"scopes"`
}

type APIClient struct {
	HTTPClient *http.Client
}

func NewAPIClient() *APIClient {
	return &APIClient{
		HTTPClient: &http.Client{},
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (c *APIClient) formatURL(server, endpoint string) string {
	server = strings.TrimSuffix(server, "/")
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	return server + endpoint
}

func (c *APIClient) do(method, server, endpoint string, body interface{}, token string) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	url := c.formatURL(server, endpoint)
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBytes, &errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("server error (status %d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	return respBytes, nil
}

// User / Token Operations

func (c *APIClient) CreateUser(server, username, password, email string) error {
	payload := map[string]string{
		"username": username,
		"password": password,
		"email":    email,
	}
	_, err := c.do("POST", server, "/api/createUser", payload, "")
	return err
}

func (c *APIClient) DeleteUser(server, username, password string) error {
	payload := map[string]string{
		"username": username,
		"password": password,
	}
	_, err := c.do("POST", server, "/api/deleteUser", payload, "")
	return err
}

type GenerateTokenResponse struct {
	Token   string `json:"token"`
	Message string `json:"message"`
}

func (c *APIClient) GenerateToken(server, username, password string, scopes []string) (string, error) {
	normalizedScopes := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			normalizedScopes = append(normalizedScopes, scope)
		}
	}

	payload := map[string]interface{}{
		"username": username,
		"password": password,
		"scopes":   normalizedScopes,
	}
	resBytes, err := c.do("POST", server, "/api/generateToken", payload, "")
	if err != nil {
		return "", err
	}
	var resp GenerateTokenResponse
	if err := json.Unmarshal(resBytes, &resp); err != nil {
		return "", err
	}
	return resp.Token, nil
}

type ListTokensResponse struct {
	Tokens []TokenInfo `json:"tokens"`
}

func (c *APIClient) ListTokens(server, token string) ([]TokenInfo, error) {
	resBytes, err := c.do("GET", server, "/api/listTokens", nil, token)
	if err != nil {
		return nil, err
	}
	var resp ListTokensResponse
	if err := json.Unmarshal(resBytes, &resp); err != nil {
		return nil, err
	}
	return resp.Tokens, nil
}

func (c *APIClient) RevokeToken(server, tokenToRevoke, password string) error {
	payload := map[string]string{
		"token":    tokenToRevoke,
		"password": password,
	}
	_, err := c.do("POST", server, "/api/revokeToken", payload, "")
	return err
}

type ValidateTokenResponse struct {
	Message string   `json:"message"`
	UserID  string   `json:"userId"`
	Scopes  []string `json:"scopes"`
}

func (c *APIClient) ValidateToken(server, tokenToValidate string) (*ValidateTokenResponse, error) {
	payload := map[string]string{
		"token": tokenToValidate,
	}
	resBytes, err := c.do("POST", server, "/api/validateToken", payload, "")
	if err != nil {
		return nil, err
	}
	var resp ValidateTokenResponse
	if err := json.Unmarshal(resBytes, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *APIClient) GrantScope(server, username, scope, token string) error {
	payload := map[string]string{
		"username": username,
		"scope":    scope,
	}
	_, err := c.do("POST", server, "/api/grantScope", payload, token)
	return err
}

type ScopesResponse struct {
	Scopes []string `json:"scopes"`
}

func (c *APIClient) GetUserScopes(server, token string) ([]string, error) {
	resBytes, err := c.do("GET", server, "/api/getUserScopes", nil, token)
	if err != nil {
		return nil, err
	}
	var resp ScopesResponse
	if err := json.Unmarshal(resBytes, &resp); err != nil {
		return nil, err
	}
	return resp.Scopes, nil
}

func (c *APIClient) GetTokenScopes(server, token string) ([]string, error) {
	resBytes, err := c.do("GET", server, "/api/getTokenScopes", nil, token)
	if err != nil {
		return nil, err
	}
	var resp ScopesResponse
	if err := json.Unmarshal(resBytes, &resp); err != nil {
		return nil, err
	}
	return resp.Scopes, nil
}

// Package Operations

type RegisterPackagePayload struct {
	Name                  string   `json:"name"`
	Version               string   `json:"version"`
	BuildSystem           string   `json:"buildSystem"`
	Dependencies          []string `json:"dependencies,omitempty"`
	CustomBuildScript     string   `json:"customBuildScript,omitempty"`
	CustomInstallScript   string   `json:"customInstallScript,omitempty"`
	CustomUninstallScript string   `json:"customUninstallScript,omitempty"`
	BuildFlags            string   `json:"buildFlags,omitempty"`
	SourceDir             string   `json:"sourceDir,omitempty"`
}

func (c *APIClient) RegisterPackage(server string, payload *RegisterPackagePayload, token string) error {
	_, err := c.do("POST", server, "/api/registerPackage", payload, token)
	return err
}

type ListPackagesResponse struct {
	Packages []Package `json:"packages"`
}

func (c *APIClient) ListPackages(server, token string) ([]Package, error) {
	resBytes, err := c.do("GET", server, "/api/listPackages", nil, token)
	if err != nil {
		return nil, err
	}
	var resp ListPackagesResponse
	if err := json.Unmarshal(resBytes, &resp); err != nil {
		return nil, err
	}
	return resp.Packages, nil
}

type GetPackageResponse struct {
	Package Package `json:"package"`
}

func (c *APIClient) GetPackage(server, name, version, token string) (*Package, error) {
	payload := map[string]string{
		"name":    name,
		"version": version,
	}
	// Note: GET endpoint with JSON body (matching the server design)
	resBytes, err := c.do("GET", server, "/api/getPackage", payload, token)
	if err != nil {
		return nil, err
	}
	var resp GetPackageResponse
	if err := json.Unmarshal(resBytes, &resp); err != nil {
		return nil, err
	}
	return &resp.Package, nil
}

func (c *APIClient) UploadPrebuilt(server, name, version, packageName, filePath, token string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("package", filepath.Base(filePath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}

	_ = writer.WriteField("name", name)
	_ = writer.WriteField("version", version)
	_ = writer.WriteField("packageName", packageName)

	err = writer.Close()
	if err != nil {
		return err
	}

	url := c.formatURL(server, "/api/uploadPrebuilt")
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBytes, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("server error (status %d): %s", resp.StatusCode, errResp.Error)
		}
		return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	return nil
}

func (c *APIClient) DownloadPrebuilt(server, name, version, token string) (io.ReadCloser, error) {
	payload := map[string]string{
		"name":    name,
		"version": version,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := c.formatURL(server, "/api/downloadPrebuilt")
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		respBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		var errResp ErrorResponse
		if err := json.Unmarshal(respBytes, &errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("server error (status %d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	return resp.Body, nil
}

func (c *APIClient) DeletePrebuilt(server, name, version, token string) error {
	payload := map[string]string{
		"name":    name,
		"version": version,
	}
	_, err := c.do("POST", server, "/api/deletePrebuilt", payload, token)
	return err
}

func (c *APIClient) DeletePackage(server, name, version, token string) error {
	payload := map[string]string{
		"name":    name,
		"version": version,
	}
	_, err := c.do("POST", server, "/api/deletePackage", payload, token)
	return err
}
