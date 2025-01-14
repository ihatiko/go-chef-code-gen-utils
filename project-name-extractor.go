package go_chef_code_gen_utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func GetPath(destination string) string {
	if destination == "" {
		d, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		return d
	}
	return destination
}

func GetProjectName(projectPath string) (string, error) {
	projectPath = GetPath(projectPath)
	goModPath := filepath.Join(projectPath, "go.mod")
	f, err := os.ReadFile(goModPath)
	if err != nil {
		return "", errors.New(fmt.Sprintf("cannot read go.mod in folder %s %v", projectPath, err))
	}
	splittedFile := strings.Split(string(f), "\n")
	if len(splittedFile) == 0 {
		return "", errors.New(fmt.Sprintf("empty go.mod file in folder %s", projectPath))
	}
	fm := strings.Replace(splittedFile[0], "module ", "", 1)
	return strings.Replace(fm, "\r", "", 1), nil
}
