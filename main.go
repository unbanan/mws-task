package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func cloneRepository(url, directory string) error {
	command := exec.Command("git", "clone", "--depth", "1", url, directory)
	return command.Run()
}

func findGoModFile(directory string) (string, error) {
	var goModPath string
	err := filepath.Walk(directory, func(path string, fi os.FileInfo, err error) error {
		if strings.HasSuffix(path, "go.mod") {
			goModPath = path
			return filepath.SkipAll
		}
		return nil
	})
	return goModPath, err
}

func getGoModInfo(directory string) ([]byte, error) {
	cmd := exec.Command("go", "mod", "edit", "-json")
	cmd.Dir = directory
	return cmd.Output()
}

var goModData struct {
	Module struct {
		Path string
	}
	Go      string
	Require []struct {
		Path    string
		Version string
	}
}

var lib struct {
	Path    string
	Version string
	Update  *struct {
		Version string
	}
}

func checkUpdates(directory string) ([]byte, error) {
	command := exec.Command("go", "list", "-u", "-m", "-json", "all")
	command.Dir = directory
	return command.Output()
}

func main() {
	var gitURL = flag.String("link", "", "GIT Repository URL")
	flag.Parse()
	if *gitURL == "" {
		log.Fatal("Git repository URL not specified via --link flag.")
	}

	repositoryDirectory, err := os.MkdirTemp("", "repo-dir-*")
	if err != nil {
		log.Fatalf("An error while creating temp directory: %v.", err)
	}
	defer os.RemoveAll(repositoryDirectory)

	if err := cloneRepository(*gitURL, repositoryDirectory); err != nil {
		log.Fatalf("An error while cloning the repository: %v.", err)
	}

	goModPath, err := findGoModFile(repositoryDirectory)
	if err != nil {
		log.Fatalf("go.mod file is not found: %v.", err)
	}
	goModDirectory := filepath.Dir(goModPath)

	goModInfo, err := getGoModInfo(goModDirectory)
	if err != nil {
		log.Fatalf("An error while reading go.mod file: %v.", err)
	}
	json.Unmarshal(goModInfo, &goModData)
	updates, err := checkUpdates(goModDirectory)
	if err != nil {
		log.Fatalf("An error while checking updates: %v.", err)
	}

	fmt.Printf("Module: %s\n", goModData.Module.Path)
	fmt.Printf("Go version: %s\n", goModData.Go)
	fmt.Println("Updates:")

	for _, line := range strings.Split(string(updates), "\n") {
		if line == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &lib); err != nil {
			continue
		}
		if lib.Update != nil {
			fmt.Printf("%s %s -> %s\n", lib.Path, lib.Version, lib.Update.Version)
		}
	}
}
