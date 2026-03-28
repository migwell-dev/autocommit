package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func checkHasCommit() (bool, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	err := cmd.Run()

	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func getUnstagedFiles() ([]string, error) {
	hasCommit, err := checkHasCommit()
	if err != nil {
		return nil, err
	}

	var files []string

	if hasCommit {
		out, err := exec.Command("git", "diff", "--name-only").Output()
		if err != nil {
			return nil, err
		}
		if trimmed := strings.TrimSpace(string(out)); trimmed != "" {
			files = append(files, strings.Split(trimmed, "\n")...)
		}
	}

	out, err := exec.Command("git", "ls-files", "--others", "--exclude-standard").Output()
	if err != nil {
		return nil, err
	}
	if trimmed := strings.TrimSpace(string(out)); trimmed != "" {
		files = append(files, strings.Split(trimmed, "\n")...)
	}

	return files, nil
}

func getStagedFiles() ([]string, error) {
	out, err := exec.Command("git", "diff", "--cached", "--name-only").Output()
	if err != nil {
		return nil, err
	}

	files := strings.Split(strings.TrimSpace(string(out)), "\n")

	if len(files) == 1 && files[0] == "" {
		return []string{}, nil
	}

	return files, nil
}

func checkForStagedFiles() (bool, error) {
	files, err := getStagedFiles()
	if err != nil {
		return false, err
	}

	if len(files) == 0 {
		return false, nil
	}

	return true, nil
}

func getFiles() ([]string, map[string]bool, error) {
	staged, err := getStagedFiles()
	if err != nil {
		return nil, nil, err
	}

	unstaged, err := getUnstagedFiles()
	if err != nil {
		return nil, nil, err
	}

	fileMap := make(map[string]bool)
	files := []string{}

	for _, f := range staged {
		files = append(files, f)
		fileMap[f] = true
	}
	for _, f := range unstaged {
		if _, exists := fileMap[f]; !exists {
			files = append(files, f)
			fileMap[f] = false
		}
	}

	return files, fileMap, nil
}

func getFileDiff(file string, isStaging bool) (string, error) {
	var out []byte
	var err error

	if isStaging {
		out, err = exec.Command("git", "diff", "--cached", file).Output()
		if err != nil {
			return "", err
		}
	} else {
		out, err = exec.Command("git", "diff", file).Output()
		if err != nil {
			return "", err
		}

		if len(out) == 0 {
			content, err := os.ReadFile(file)
			if err != nil {
				return "", fmt.Errorf("could not read file %s: %w", file, err)
			}
			out = content
		}
	}

	return string(out), nil
}

func addFilesToStaging(files []string) error {
	if len(files) == 0 {
		return nil
	}

	args := append([]string{"add"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func runCommit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
