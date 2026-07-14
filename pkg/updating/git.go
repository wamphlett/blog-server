package updating

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/attribute"
)

func (u *Updater) updateFromRemote(ctx context.Context, forceFresh bool) error {
	ctx, span := tracer.Start(ctx, "update.UpdateFromRemote")
	defer span.End()

	// if forcing the update, then remove the current content directory which
	// will initiate a clone
	if forceFresh {
		if err := os.RemoveAll(u.path); err != nil {
			return err
		}
	}

	// if the content directory does not exist, clone the given repo
	if _, err := os.Stat(u.path); os.IsNotExist(err) {
		span.SetAttributes(attribute.String("action", "clone"))
		if err := u.clone(ctx); err != nil {
			return err
		}

		// only attempt to change branch if one has been configured
		if u.branch != "" {
			if err := u.checkout(ctx); err != nil {
				return err
			}
		}

		return nil
	}

	// only attempt to change branch if one has been configured, doing this
	// before pulling ensures the pull always updates the intended branch
	if u.branch != "" {
		if err := u.checkout(ctx); err != nil {
			return err
		}
	}

	span.SetAttributes(attribute.String("action", "pull"))
	return u.pull(ctx)
}

// clone does a git clone from the remote repository
func (u *Updater) clone(ctx context.Context) error {
	_, span := tracer.Start(ctx, "update.Clone")
	span.SetAttributes(attribute.String("repo", u.repo))
	defer span.End()

	slog.InfoContext(ctx, "cloning repository", "repo", u.repo)
	cmd := exec.Command("git", "clone", u.repo, u.path)
	if out, err := cmd.CombinedOutput(); err != nil {
		slog.ErrorContext(ctx, "git clone failed", "repo", u.repo, "output", string(out), "error", err)
		return err
	}
	slog.InfoContext(ctx, "repository cloned", "repo", u.repo)
	return nil
}

// checkout changes to the configured branch
func (u *Updater) checkout(ctx context.Context) error {
	_, span := tracer.Start(ctx, "update.Checkout")
	span.SetAttributes(attribute.String("repo", u.repo), attribute.String("branch", u.branch))
	defer span.End()

	slog.InfoContext(ctx, "checking out branch", "repo", u.repo, "branch", u.branch)
	cmd := exec.Command("git", "checkout", u.branch)
	cmd.Env = []string{
		fmt.Sprintf("GIT_DIR=%s/.git", u.path),
		fmt.Sprintf("GIT_WORK_TREE=%s", u.path),
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		slog.ErrorContext(ctx, "git checkout failed", "repo", u.repo, "branch", u.branch, "output", string(out), "error", err)
		return err
	}
	slog.InfoContext(ctx, "branch checked out", "repo", u.repo, "branch", u.branch)
	return nil
}

// fileCreatedAt returns the unix timestamp of the commit that first added
// filePath to the repository rooted at u.path. It returns 0 if the timestamp
// cannot be determined, e.g. the content directory isn't a git repository or
// the file isn't tracked - callers should treat that as "unknown" rather
// than a real creation time.
func (u *Updater) fileCreatedAt(ctx context.Context, filePath string) int64 {
	_, span := tracer.Start(ctx, "update.FileCreatedAt")
	defer span.End()

	relPath, err := filepath.Rel(u.path, filePath)
	if err != nil {
		return 0
	}

	cmd := exec.Command("git", "-C", u.path, "log", "--follow", "--format=%at", "--", relPath)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}

	commitTimes := strings.Fields(string(out))
	if len(commitTimes) == 0 {
		return 0
	}

	// git log lists newest first, so the last entry is the file's first commit
	createdAt, err := strconv.ParseInt(commitTimes[len(commitTimes)-1], 10, 64)
	if err != nil {
		return 0
	}

	return createdAt
}

// pull does a git pull from the remote repository
func (u *Updater) pull(ctx context.Context) error {
	_, span := tracer.Start(ctx, "update.Pull")
	span.SetAttributes(attribute.String("repo", u.repo))
	defer span.End()

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
