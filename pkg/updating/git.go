package updating

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

func (u *Updater) updateFromRemote(ctx context.Context, forceFresh bool) error {
	// if forcing the update, then remove the current content directory which
	// will initiate a clone
	if forceFresh {
		if err := os.RemoveAll(u.path); err != nil {
			return err
		}
	}

	// if the content directory does not exist, clone the given repo
	if _, err := os.Stat(u.path); os.IsNotExist(err) {
		if err := u.clone(ctx); err != nil {
			return err
		}
		return nil
	}
	// pull recent changes
	return u.pull(ctx)
}

// clone does a git clone from the remote repository
func (u *Updater) clone(ctx context.Context) error {
	slog.InfoContext(ctx, "cloning repository", "repo", u.repo)
	cmd := exec.Command("git", "clone", u.repo, u.path)
	if out, err := cmd.CombinedOutput(); err != nil {
		slog.ErrorContext(ctx, "git clone failed", "repo", u.repo, "output", string(out), "error", err)
		return err
	}
	slog.InfoContext(ctx, "repository cloned", "repo", u.repo)
	return nil
}

// pull does a git pull from the remote repository
func (u *Updater) pull(ctx context.Context) error {
	slog.InfoContext(ctx, "pulling changes from repository", "repo", u.repo)
	cmd := exec.Command("git", "pull")
	cmd.Env = []string{
		fmt.Sprintf("GIT_DIR=%s/.git", u.path),
		fmt.Sprintf("GIT_WORK_TREE=%s", u.path),
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		slog.ErrorContext(ctx, "git pull failed", "repo", u.repo, "output", string(out), "error", err)
		return err
	}
	slog.InfoContext(ctx, "repository pulled", "repo", u.repo)
	return nil
}
