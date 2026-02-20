package gitutil

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func EnsureRepo(ctx context.Context, workdir string) error {
	_, err := runGit(ctx, workdir, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return errors.New("current directory is not a git repository")
	}
	return nil
}

func EnsureClean(ctx context.Context, workdir string) error {
	out, err := runGit(ctx, workdir, "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "" {
		return errors.New("git working tree is dirty; please commit, stash (git stash -u), or remove untracked/modified files before running devspec")
	}
	return nil
}

func Checkout(ctx context.Context, workdir, branch string) error {
	_, err := runGit(ctx, workdir, "checkout", branch)
	return err
}

func PullFFOnly(ctx context.Context, workdir string) error {
	_, err := runGit(ctx, workdir, "pull", "--ff-only")
	if err != nil && strings.Contains(err.Error(), "There is no tracking information") {
		// Local-only repo, safely ignore
		return nil
	}
	return err
}

func CreateBranch(ctx context.Context, workdir, branch string) error {
	_, err := runGit(ctx, workdir, "checkout", "-b", branch)
	return err
}

func CurrentBranch(ctx context.Context, workdir string) (string, error) {
	out, err := runGit(ctx, workdir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func RepoRoot(ctx context.Context, workdir string) (string, error) {
	out, err := runGit(ctx, workdir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return filepath.Clean(strings.TrimSpace(out)), nil
}

func Diff(ctx context.Context, workdir string) (string, error) {
	return runGit(ctx, workdir, "diff")
}

func DiffStat(ctx context.Context, workdir string) (string, error) {
	return runGit(ctx, workdir, "diff", "--stat")
}

func ChangedFiles(ctx context.Context, workdir string) ([]string, error) {
	out, err := runGit(ctx, workdir, "diff", "--name-only")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 1 && strings.TrimSpace(lines[0]) == "" {
		return nil, nil
	}
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}

func RepoTree(ctx context.Context, workdir string) (string, error) {
	return runGit(ctx, workdir, "ls-tree", "-r", "--name-only", "HEAD")
}

func AddAll(ctx context.Context, workdir string) error {
	_, err := runGit(ctx, workdir, "add", "-A")
	return err
}

func Commit(ctx context.Context, workdir, message string) error {
	_, err := runGit(ctx, workdir, "commit", "-m", message)
	return err
}

func Push(ctx context.Context, workdir, branch string) error {
	_, err := runGit(ctx, workdir, "push", "-u", "origin", branch)
	return err
}

func DiffLineCount(diff string) int {
	var count int
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			count++
		}
	}
	return count
}

func runGit(ctx context.Context, workdir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
