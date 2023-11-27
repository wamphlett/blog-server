package updating

import (
	"fmt"
	"os"
	"os/exec"

	log "unknwon.dev/clog/v2"
)

func (u *Updater) updateFromRemote(forceFresh bool) error {
	// if forcing the update, then remove the current content directory which
	// will initiate a clone
	if forceFresh {
		if err := os.RemoveAll(u.path); err != nil {
			return err
		}
	}

	// if the content directory does not exist, clone the given repo
	if _, err := os.Stat(u.path); os.IsNotExist(err) {
		if err := u.clone(); err != nil {
			return err
		}
		return nil
	}
	// pull recent changes
	return u.pull()
}

// clone does a git clone from the remote repository
func (u *Updater) clone() error {
	log.Info("cloning %s", u.repo)
	// clone the repo
	cmd := exec.Command("git", "clone", u.repo, u.path)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// pull does a git pull from the remote repository
func (u *Updater) pull() error {
	log.Info("pulling changes from %s", u.repo)
	// Get the changes from the remote repo
	cmd := exec.Command("git", "pull")
	cmd.Env = []string{
		fmt.Sprintf("GIT_DIR=%s/.git", u.path),
		fmt.Sprintf("GIT_WORK_TREE=%s", u.path),
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
