package crush

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
)

const appName = "gh-slop"
const crushSubDir = "crush"

//go:embed crush.json
var defaultConfig []byte

//go:embed skills/slop-detect/SKILL.md
var slopDetectSkill []byte

func CrushDir() (string, error) {
	dir, err := xdgConfigHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appName, crushSubDir), nil
}

func xdgConfigHome() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir, nil
	}
	return os.UserConfigDir()
}

func EnsureConfig() error {
	crushDir, err := CrushDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(crushDir, "crush.json")
	if _, err := os.Stat(configPath); err != nil {
		if err := os.MkdirAll(crushDir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			return err
		}
	}

	skillsDir := filepath.Join(crushDir, "skills", "slop-detect")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), slopDetectSkill, 0644)
}

func Run(ctx context.Context) error {
	return runCrush(ctx, nil)
}

func RunDetect(ctx context.Context, repoList []string) error {
	if len(repoList) == 0 {
		repo, err := repository.Current()
		if err != nil {
			return fmt.Errorf("no repository specified and not in a git repo: %w", err)
		}
		repoList = []string{repo.Owner + "/" + repo.Name}
	}
	prompt := "detect slop in " + strings.Join(repoList, ", ")
	return runCrush(ctx, []string{"run", prompt})
}

func runCrush(ctx context.Context, args []string) error {
	if err := EnsureConfig(); err != nil {
		return err
	}

	crushDir, err := CrushDir()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "crush", args...)
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("CRUSH_GLOBAL_CONFIG=%s", crushDir),
		fmt.Sprintf("CRUSH_SKILLS_DIR=%s", filepath.Join(crushDir, "skills")),
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
