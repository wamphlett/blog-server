package updating

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	log "unknwon.dev/clog/v2"
)

type Metrics interface {
	ContentUpdated(startTime time.Time)
}

type Updater struct {
	path    string
	repo    string
	metrics Metrics
}

func New(repoUrl, contentPath string, refreshInterval time.Duration, metrics Metrics, onRefresh func()) (*Updater, error) {
	u := &Updater{
		path:    contentPath,
		repo:    repoUrl,
		metrics: metrics,
	}
	// update immediately
	if err := u.Update(true); err != nil {
		return nil, err
	}
	// schedule further updates on the defined interval
	go scheduleUpdates(refreshInterval, func() {
		if err := u.Update(false); err != nil {
			log.Error("error when updating content", err)
		}
		onRefresh()
	})

	log.Info("updater configured to refresh content every %.0f seconds", refreshInterval.Seconds())

	return u, nil
}

func (u *Updater) Update(forceFresh bool) error {
	startTime := time.Now()
	defer u.metrics.ContentUpdated(startTime)
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
	log.Info("cloning %s", u.repo)
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

func scheduleUpdates(interval time.Duration, f func()) {
	for range time.Tick(interval) {
		f()
	}
}
