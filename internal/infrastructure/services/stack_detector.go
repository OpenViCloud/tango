package services

import (
	"os"
	"path/filepath"
)

type DetectedStack string

const (
	StackGo         DetectedStack = "go"
	StackNode       DetectedStack = "node"
	StackPython     DetectedStack = "python"
	StackRust       DetectedStack = "rust"
	StackJava       DetectedStack = "java"
	StackDotNet     DetectedStack = "dotnet"
	StackDockerfile DetectedStack = "dockerfile" // repo already has a Dockerfile
	StackUnknown    DetectedStack = "unknown"
)

// DetectStack inspects the directory and returns the detected stack.
// If a Dockerfile already exists it returns StackDockerfile.
func DetectStack(dir string) DetectedStack {
	check := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	glob := func(pattern string) bool {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		return err == nil && len(matches) > 0
	}

	if check("Dockerfile") {
		return StackDockerfile
	}
	if check("go.mod") {
		return StackGo
	}
	if check("package.json") {
		return StackNode
	}
	if check("requirements.txt") || check("pyproject.toml") || check("setup.py") {
		return StackPython
	}
	if check("Cargo.toml") {
		return StackRust
	}
	if check("pom.xml") || check("build.gradle") || check("build.gradle.kts") {
		return StackJava
	}
	if glob("*.csproj") || glob("*.fsproj") || glob("*.sln") {
		return StackDotNet
	}
	return StackUnknown
}
