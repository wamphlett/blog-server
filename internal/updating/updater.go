package updating

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

type Updater struct {
	path string
	repo string
}

func New(repoUrl, contentPath string, refreshInterval time.Duration, onRefresh func()) (*Updater, error) {
	u := &Updater{
		path: contentPath,
		repo: repoUrl,
	}
	// update immediately
	if err := u.Update(true); err != nil {
		return nil, err
	}
	// schedule further updates on the defined interval
	go scheduleUpdates(refreshInterval, func() {
		if err := u.Update(false); err != nil {
			log.Println("ERROR when updating")
		}
		onRefresh()
	})
	return u, nil
}

func (u *Updater) Update(forceFresh bool) error {
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

func (u *Updater) clone() error {
	log.Printf("cloning %s", u.repo)
	// clone the repo
	cmd := exec.Command("git", "clone", u.repo, u.path)
	// cmd.Stdout = s.l.Info
	// cmd.Stderr = s.l.Info
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (u *Updater) pull() error {
	log.Printf("pulling changes from %s", u.repo)
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

func scheduleUpdates(interval time.Duration, f func()) {
	for range time.Tick(interval) {
		f()
	}
}
