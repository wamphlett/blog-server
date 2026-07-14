package updating

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runGit runs a git command in dir, failing the test on error
func runGit(t *testing.T, dir string, env []string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, out)
}

// commitAt commits path in dir with a fixed author/committer date so the test
// isn't sensitive to when it runs
func commitAt(t *testing.T, dir, path string, at time.Time) {
	t.Helper()
	runGit(t, dir, nil, "add", path)
	dateEnv := []string{
		"GIT_AUTHOR_DATE=" + at.Format(time.RFC3339),
		"GIT_COMMITTER_DATE=" + at.Format(time.RFC3339),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	}
	runGit(t, dir, dateEnv, "commit", "-m", "commit "+path)
}

// verifies that fileCreatedAt returns the timestamp of the commit that first added
// the file, not any later commit that modified it
func TestFileCreatedAt_ReturnsFirstCommitTimestamp(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, nil, "init")

	createdAt := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 3, 1, 9, 0, 0, 0, time.UTC)

	articlePath := filepath.Join(dir, "article.md")
	require.NoError(t, os.WriteFile(articlePath, []byte("first version"), 0644))
	commitAt(t, dir, "article.md", createdAt)

	require.NoError(t, os.WriteFile(articlePath, []byte("updated version"), 0644))
	commitAt(t, dir, "article.md", updatedAt)

	u := &Updater{path: dir}
	got := u.fileCreatedAt(context.Background(), articlePath)

	assert.Equal(t, createdAt.Unix(), got)
}

// verifies that fileCreatedAt returns 0 when the content directory isn't a git
// repository, rather than erroring
func TestFileCreatedAt_ReturnsZeroWhenNotAGitRepo(t *testing.T) {
	dir := t.TempDir()
	articlePath := filepath.Join(dir, "article.md")
	require.NoError(t, os.WriteFile(articlePath, []byte("content"), 0644))

	u := &Updater{path: dir}
	got := u.fileCreatedAt(context.Background(), articlePath)

	assert.Equal(t, int64(0), got)
}

// verifies that fileCreatedAt returns 0 for a file that exists on disk but has never
// been committed
func TestFileCreatedAt_ReturnsZeroForUntrackedFile(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, nil, "init")

	articlePath := filepath.Join(dir, "untracked.md")
	require.NoError(t, os.WriteFile(articlePath, []byte("content"), 0644))

	u := &Updater{path: dir}
	got := u.fileCreatedAt(context.Background(), articlePath)

	assert.Equal(t, int64(0), got)
}
