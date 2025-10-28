package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestLoadPackages(t *testing.T) {
	tests := []struct {
		name        string
		patterns    []string
		wantErr     bool
		checkResult func(t *testing.T, pkgs []*packages.Package)
	}{
		{
			name:     "load test package",
			patterns: []string{"./test"},
			wantErr:  false,
			checkResult: func(t *testing.T, pkgs []*packages.Package) {
				if len(pkgs) == 0 {
					t.Error("expected at least one package")
				}
			},
		},
		{
			name:     "invalid pattern",
			patterns: []string{"./nonexistent"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgs, err := loadPackages(tt.patterns)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadPackages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.checkResult != nil && !tt.wantErr {
				tt.checkResult(t, pkgs)
			}
		})
	}
}

func TestFindStructsWithResetTag(t *testing.T) {
	pkgs, err := loadPackages([]string{"./test"})
	if err != nil {
		t.Fatalf("failed to load packages: %v", err)
	}

	result := findStructsWithResetTag(pkgs)

	if len(result) == 0 {
		t.Fatal("expected to find at least one package with reset tag")
	}

	found := false
	for _, d := range result {
		if d.PkgName == "test" {
			found = true
			if len(d.StructNames) != 1 {
				t.Errorf("test package: expected 1 struct, got %d", len(d.StructNames))
			}
			if len(d.StructNames) > 0 && d.StructNames[0].Name != "Test" {
				t.Errorf("test package: expected Test, got %s", d.StructNames[0].Name)
			}
		}
	}

	if !found {
		t.Error("expected to find test package")
	}
}

func TestGenerateResetFiles(t *testing.T) {
	tmpDir := t.TempDir()

	testData := map[string]data{
		tmpDir: {
			PkgName: "testpkg",
			StructNames: []structInfo{
				{Name: "TestStruct", ReceiverName: "t"},
			},
		},
	}

	err := generateResetFiles(testData)
	if err != nil {
		t.Fatalf("generateResetFiles() error = %v", err)
	}

	genFile := filepath.Join(tmpDir, "reset.gen.go")
	content, err := os.ReadFile(genFile)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "package testpkg") {
		t.Error("generated file should contain package declaration")
	}
	if !strings.Contains(contentStr, "func (t *TestStruct) Reset()") {
		t.Error("generated file should contain Reset method for TestStruct")
	}
	if !strings.Contains(contentStr, "Code generated") {
		t.Error("generated file should contain generation comment")
	}
}

func TestGenerate(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "./test"}

	main()

	genFile := "test/reset.gen.go"
	if _, err := os.Stat(genFile); os.IsNotExist(err) {
		t.Fatalf("expected file %s to be generated", genFile)
	}

	content, err := os.ReadFile(genFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", genFile, err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Code generated") {
		t.Error("generated file should contain generation comment")
	}
	if !strings.Contains(contentStr, "package test") {
		t.Error("generated file should have correct package name")
	}
	if !strings.Contains(contentStr, "func (t *Test) Reset()") {
		t.Error("generated file should contain Reset method")
	}
}
