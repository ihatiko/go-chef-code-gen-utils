package go_chef_code_gen_utils

import (
	"bufio"
	"errors"
	"fmt"
	"golang.org/x/mod/semver"

	"io"
	"log/slog"
	"net/http"
	"strings"
)

type Updater struct {
	proxies []string
}

func NewUpdater(proxies []string) *Updater {
	return &Updater{proxies: proxies}
}

func (u *Updater) AutoUpdate(packageName string) {
	splittedPackage := strings.Split(packageName, "/")
	packageUrl := splittedPackage[len(splittedPackage)-1]
	composer := NewExecutor()
	_, err := composer.ExecDefaultCommand(packageUrl)
	if err != nil {
		slog.Info("does not found module", slog.String("module", packageName))
		slog.Info("try installing module", slog.String("module", packageName))
		lastVersion, err := u.GetLastVersion(packageName)
		if err != nil {
			slog.Error("Failed to get last version", slog.Any("error", err))
			return
		}
		installInstruction := fmt.Sprintf("%s@%s", packageName, lastVersion)
		command := fmt.Sprintf("go install %s", installInstruction)
		slog.Info("Executing", slog.String("command", command))
		_, err = composer.ExecDefaultCommand(command)
		if err != nil {
			slog.Error("error executing composer", slog.Any("error", err))
			return
		}
		return
	}
	currentVersion, err := composer.ExecDefaultCommand(fmt.Sprintf("%s version", packageUrl))
	if err != nil {
		slog.Error("Failed to execute composer", slog.Any("error", err), slog.String("package", packageUrl))
		return
	}
	lastVersion, err := u.GetLastVersion(packageName)
	if err != nil {
		slog.Error("Failed to get last version", slog.Any("error", err))
		return
	}
	formattedCurrentVersion := strings.ReplaceAll(currentVersion.String(), "\n", "")
	if lastVersion == formattedCurrentVersion {
		slog.Info("actual", slog.String("package", packageName), slog.String("version", formattedCurrentVersion))
		return
	}
	if !semver.IsValid(formattedCurrentVersion) {
		formattedCurrentVersion = "Unknown"
	}
	slog.Info("try update", slog.String("current-version", formattedCurrentVersion), slog.String("last-version", lastVersion))
	installInstruction := fmt.Sprintf("%s@%s", packageName, lastVersion)
	command := fmt.Sprintf("go install %s", installInstruction)

	slog.Info("Executing", slog.String("command", command))
	_, err = composer.ExecDefaultCommand(command)
	if err != nil {
		slog.Error("error executing composer", slog.Any("error", err))
		return
	}
}

func (u *Updater) GetLastVersion(packageName string) (string, error) {
	for _, proxy := range u.proxies {
		fullPathName := fmt.Sprintf("%s/%s/@v/list", proxy, packageName)
		response, err := http.Get(fullPathName)
		if err != nil || response.StatusCode != 200 {
			continue
		}

		reader := bufio.NewReader(response.Body)

		bytes, err := reader.ReadBytes(0)
		if err != nil && err != io.EOF {
			continue
		}
		if len(bytes) == 0 {
			continue
		}
		versions := strings.Split(string(bytes), "\n")

		semver.Sort(versions)
		lastVersion := versions[len(versions)-1]
		return lastVersion, nil
	}
	return "", errors.New("no version found")
}
