package crush

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	if err := EnsureConfig(); err != nil {
		return err
	}

	crushDir, err := CrushDir()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "crush")
	cmd.Env = append(
		os.Environ(),
		// TODO add GH_REPO if currently in a repo (this needs to be acknowledged in slop)
		fmt.Sprintf("CRUSH_GLOBAL_CONFIG=%s", crushDir),
		fmt.Sprintf("CRUSH_SKILLS_DIR=%s", filepath.Join(crushDir, "skills")),
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
