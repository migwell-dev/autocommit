package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/atotto/clipboard"
	"net/http"
	"os/exec"
	"strings"
)

var providers = []string{"ollama", "claude", "codex", "copy", "skip"}

func buildPrompt(diff, commitType string) string {
	return fmt.Sprintf(`You are a git commit message generator. Given the following diff and commit type, write a commit message description only — do NOT include the type prefix, just the description after the colon.

Commit type: %s
Reply with only the description, e.g. "add login page" not "feat: add login page".
Carefully inspect the diff and generate the main idea of the changes that were performed.
1 to 2 lines only.

Diff:
%s`, commitType, diff)
}

func savePromptToClipboard(prompt string) (string, error) {
	err := clipboard.WriteAll(prompt)
	if err != nil {
		return "failed to save prompt to clipboard", err
	}
	return "", nil
}

func parseGeneratedMessage(raw, commitType string) string {
	raw = strings.TrimSpace(raw)
	prefix := commitType + ": "

	if strings.HasPrefix(strings.ToLower(raw), strings.ToLower(prefix)) {
		raw = strings.TrimSpace(raw[len(prefix):])
	} else if idx := strings.Index(raw, ": "); idx != -1 && idx < 20 {
		raw = strings.TrimSpace(raw[idx+2:])
	}

	if len(raw) >= 2 {
		if (raw[0] == '"' && raw[len(raw)-1] == '"') || (raw[0] == '\'' && raw[len(raw)-1] == '\'') {
			raw = raw[1 : len(raw)-1]
		}
	}

	return raw
}

func getOllamaModels() ([]string, error) {
	out, err := exec.Command("ollama", "list").Output()
	if err != nil {
		return nil, fmt.Errorf("ollama not found: %w", err)
	}
	var models []string
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			models = append(models, fields[0])
		}
	}
	return models, nil
}

func generateWithOllama(model, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	})
	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("ollama not running: %w", err)
	}
	defer resp.Body.Close()
	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Response), nil
}

func generateWithClaude(prompt string) (string, error) {
	out, err := exec.Command("claude", "-p", prompt).Output()
	if err != nil {
		return "", fmt.Errorf("claude CLI not found or not authenticated: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func generateWithCodex(prompt string) (string, error) {
	out, err := exec.Command("codex", "-p", prompt).Output()
	if err != nil {
		return "", fmt.Errorf("codex CLI not found: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func generateCommitMessage(provider string, stagedFiles []string, fileMap map[string]bool, ollamaModel string, commitType string) (string, error) {
	var diffBuilder strings.Builder
	for _, f := range stagedFiles {
		diff, err := getFileDiff(f, fileMap[f])
		if err != nil {
			continue
		}
		fmt.Fprintf(&diffBuilder, "=== %s ===\n%s\n", f, diff)
	}

	prompt := buildPrompt(diffBuilder.String(), commitType)

	switch provider {
	case "ollama":
		return generateWithOllama(ollamaModel, prompt)
	case "claude":
		return generateWithClaude(prompt)
	case "codex":
		return generateWithCodex(prompt)
	case "copy":
		return savePromptToClipboard(prompt)
	default:
		return "", nil
	}
}
