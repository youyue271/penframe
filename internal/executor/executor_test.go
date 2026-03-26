package executor

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

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
	command := `nmap -sV -oN "C:\Temp\nmap.txt" -oX /tmp/nmap.xml -oA report/all target.local`
	got := extractDeclaredOutputFiles(command)
	want := []string{
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
