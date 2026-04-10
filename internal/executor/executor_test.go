package executor

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"penframe/internal/domain"
)

func TestLocalExecutorRunsShellCommand(t *testing.T) {
	execImpl := NewLocalExecutor()
	result, err := execImpl.Execute(
		context.Background(),
		domain.WorkflowNode{ID: "shell-node"},
		domain.ToolDefinition{},
		nil,
		"printf 'hello-local'",
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got := strings.TrimSpace(result.Stdout); got != "hello-local" {
		t.Fatalf("expected stdout hello-local, got %q", got)
	}
	if result.Metadata["launcher"] != "shell" {
		t.Fatalf("expected launcher shell, got %#v", result.Metadata["launcher"])
	}
	if result.Metadata["powershell"] != false {
		t.Fatalf("expected powershell false, got %#v", result.Metadata["powershell"])
	}
	if result.Metadata["shell_path"] == "" {
		t.Fatal("expected shell_path metadata to be populated")
	}
}

func TestLocalExecutorReturnsErrorForFailingCommand(t *testing.T) {
	execImpl := NewLocalExecutor()
	result, err := execImpl.Execute(
		context.Background(),
		domain.WorkflowNode{ID: "failing-node"},
		domain.ToolDefinition{},
		nil,
		"echo 'before-fail' && false",
	)
	if err == nil {
		t.Fatal("expected Execute to return an error")
	}
	if !strings.Contains(result.Stdout, "before-fail") {
		t.Fatalf("expected stdout to contain before-fail, got %q", result.Stdout)
	}
	if _, ok := result.Metadata["exit_code"]; !ok {
		t.Fatal("expected exit_code metadata to be populated")
	}
}

func TestShouldUsePowerShell(t *testing.T) {
	cases := []struct {
		command string
		want    bool
	}{
		{command: "\"/mnt/h/tools/Penetration/tools/01 scan/Nmap/nmap.exe\" -sV", want: true},
		{command: "& \"C:/tools/fscan.exe\" -h 127.0.0.1", want: true},
		{command: "/usr/bin/nmap -sV", want: false},
		{command: "nmap -sV", want: false},
	}
	for _, tc := range cases {
		got := shouldUsePowerShell(tc.command)
		if got != tc.want {
			t.Fatalf("shouldUsePowerShell(%q) = %v, want %v", tc.command, got, tc.want)
		}
	}
}

func TestExtractDeclaredOutputFiles(t *testing.T) {
	command := `curl -D /tmp/headers.txt --output /tmp/body.html && nmap -sV -oN "C:\Temp\nmap.txt" -oX /tmp/nmap.xml -oA report/all target.local`
	got := extractDeclaredOutputFiles(command)
	want := []string{
		`/tmp/headers.txt`,
		`/tmp/body.html`,
		`C:\Temp\nmap.txt`,
		`/tmp/nmap.xml`,
		`report/all.nmap`,
		`report/all.xml`,
		`report/all.gnmap`,
	}
	if !slices.Equal(got, want) {
		t.Fatalf("extractDeclaredOutputFiles() = %#v, want %#v", got, want)
	}
}

func TestReadDeclaredOutputFilesReadsContent(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "fscan.json")
	if err := os.WriteFile(file, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	files := readDeclaredOutputFiles("fscan -u https://x -o " + file)
	if len(files) != 1 {
		t.Fatalf("expected 1 output file entry, got %d", len(files))
	}
	entry := files[0]
	if entry["path"] != file {
		t.Fatalf("expected path %q, got %#v", file, entry["path"])
	}
	if content, _ := entry["content"].(string); content != `{"ok":true}` {
		t.Fatalf("expected content to be read, got %#v", entry["content"])
	}
}

func TestCleanDeclaredOutputFilesRemovesPreviousResult(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "nmap-quick.txt")
	if err := os.WriteFile(file, []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cleanDeclaredOutputFiles([]string{file})
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("expected output file to be removed, got err=%v", err)
	}
}

func TestNewLocalExecutorUsesCurrentShellEnv(t *testing.T) {
	t.Setenv("SHELL", "/bin/custom-shell")

	execImpl := NewLocalExecutor()
	if execImpl.shellPath != "/bin/custom-shell" {
		t.Fatalf("expected shellPath /bin/custom-shell, got %q", execImpl.shellPath)
	}
}

func TestLocalExecutorRespectsNodeShellOverride(t *testing.T) {
	root := t.TempDir()
	logFile := filepath.Join(root, "shell.log")
	wrapperPath := filepath.Join(root, "shell-wrapper.sh")
	wrapper := "#!/bin/sh\nprintf '%s\\n' \"$0 $1 $2\" > \"" + logFile + "\"\nexec /bin/sh \"$@\"\n"
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	execImpl := LocalExecutor{
		shellPath: "/bin/sh",
		timeout:   30 * time.Minute,
	}
	result, err := execImpl.Execute(
		context.Background(),
		domain.WorkflowNode{ID: "override-shell", Shell: wrapperPath},
		domain.ToolDefinition{},
		nil,
		"printf 'hello-override'",
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got := strings.TrimSpace(result.Stdout); got != "hello-override" {
		t.Fatalf("expected stdout hello-override, got %q", got)
	}
	if got := result.Metadata["shell_path"]; got != wrapperPath {
		t.Fatalf("expected shell_path %q, got %#v", wrapperPath, got)
	}

	logData, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	logText := string(logData)
	if !strings.Contains(logText, wrapperPath) || !strings.Contains(logText, "-lc") {
		t.Fatalf("expected wrapper log to mention shell path and -lc, got %q", logText)
	}
}

func TestLocalExecutorAppliesNodeEnvOverrides(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:13579")

	execImpl := LocalExecutor{
		shellPath: "/bin/sh",
		timeout:   30 * time.Minute,
	}
	result, err := execImpl.Execute(
		context.Background(),
		domain.WorkflowNode{
			ID:  "env-node",
			Env: map[string]any{"HTTP_PROXY": "", "CUSTOM_FLAG": "hello"},
		},
		domain.ToolDefinition{},
		nil,
		"printf '%s|%s' \"$HTTP_PROXY\" \"$CUSTOM_FLAG\"",
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got := strings.TrimSpace(result.Stdout); got != "|hello" {
		t.Fatalf("expected stdout |hello, got %q", got)
	}
	keys, _ := result.Metadata["env_keys"].([]string)
	if !slices.Equal(keys, []string{"CUSTOM_FLAG", "HTTP_PROXY"}) {
		t.Fatalf("expected env_keys to be recorded, got %#v", result.Metadata["env_keys"])
	}
}

func TestHTTPExecutorWritesHeadersAndBody(t *testing.T) {
	root := t.TempDir()
	headersFile := filepath.Join(root, "headers.txt")
	bodyFile := filepath.Join(root, "body.html")

	execImpl := HTTPExecutor{
		client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusTemporaryRedirect,
					Status:     "307 Temporary Redirect",
					Proto:      "HTTP/1.1",
					ProtoMajor: 1,
					ProtoMinor: 1,
					Header: http.Header{
						"Location":     []string{"/apps"},
						"X-Powered-By": []string{"Next.js"},
					},
					Body: io.NopCloser(strings.NewReader("<title>Dify</title>")),
				}, nil
			}),
		},
	}
	result, err := execImpl.Execute(
		context.Background(),
		domain.WorkflowNode{ID: "http-node"},
		domain.ToolDefinition{},
		map[string]any{
			"url":          "https://demo.example:3000",
			"headers_file": headersFile,
			"body_file":    bodyFile,
			"max_time":     "5",
		},
		"",
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got := result.Metadata["launcher"]; got != "http" {
		t.Fatalf("expected launcher http, got %#v", got)
	}
	if got := result.Metadata["status_code"]; got != http.StatusTemporaryRedirect {
		t.Fatalf("expected status code %d, got %#v", http.StatusTemporaryRedirect, got)
	}

	headersData, err := os.ReadFile(headersFile)
	if err != nil {
		t.Fatalf("ReadFile headers returned error: %v", err)
	}
	if !strings.Contains(string(headersData), "HTTP/1.1 307 Temporary Redirect") {
		t.Fatalf("expected status line in headers, got %q", string(headersData))
	}
	if !strings.Contains(string(headersData), "Location: /apps") {
		t.Fatalf("expected location header, got %q", string(headersData))
	}

	bodyData, err := os.ReadFile(bodyFile)
	if err != nil {
		t.Fatalf("ReadFile body returned error: %v", err)
	}
	if string(bodyData) != "<title>Dify</title>" {
		t.Fatalf("unexpected body content %q", string(bodyData))
	}
}

func TestProxyURLFromEnvPrefersHTTPThenAllProxy(t *testing.T) {
	target, err := url.Parse("https://demo.example")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	proxyURL, err := proxyURLFromEnv(target, map[string]string{
		"HTTP_PROXY": "http://127.0.0.1:8080",
		"ALL_PROXY":  "socks5://127.0.0.1:1080",
	})
	if err != nil {
		t.Fatalf("proxyURLFromEnv returned error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://127.0.0.1:8080" {
		t.Fatalf("expected HTTP proxy to win, got %#v", proxyURL)
	}
}

func TestProxyURLFromEnvFallsBackToAllProxy(t *testing.T) {
	target, err := url.Parse("http://demo.example")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	proxyURL, err := proxyURLFromEnv(target, map[string]string{
		"ALL_PROXY": "socks5://127.0.0.1:1080",
	})
	if err != nil {
		t.Fatalf("proxyURLFromEnv returned error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "socks5://127.0.0.1:1080" {
		t.Fatalf("expected ALL_PROXY fallback, got %#v", proxyURL)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
