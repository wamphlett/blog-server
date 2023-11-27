package updating

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/pkg/errors"
	"github.com/wamphlett/blog-server/pkg/model"
	log "unknwon.dev/clog/v2"
)

type Database interface {
	SetTopics(topic *model.Topic) error
	SetArticle(article *model.Article) error
}

type Receiver func(topic []*model.Topic, article []*model.Article)

// Metrics defines the metrics used by the updater
type Metrics interface {
	ContentUpdated(startTime time.Time)
}

type Reader interface {
	LoadTopicFromFile(topicFilePath string) *model.Topic
	LoadArticleFromFile(articleFilePath, topicSlug string) *model.Article
}

// Updater defines a new updater
type Updater struct {
	path      string
	topicFile string
	repo      string

	metrics Metrics
	reader  Reader

	callbacks []func()

	receivers []Receiver

	refreshInterval time.Duration

	fileChecksums map[string]string
}

// Option defines the function used to set options
type Option func(*Updater)

// WithCallback defines a callback to use after each update
func WithCallback(function func()) Option {
	return func(u *Updater) {
		u.callbacks = append(u.callbacks, function)
	}
}

func WithRemoteRepository(repo string) Option {
	return func(u *Updater) {
		u.repo = repo
	}
}

func WithRefreshInterval(refreshInterval time.Duration) Option {
	return func(u *Updater) {
		u.refreshInterval = refreshInterval
	}
}

func WithReceiver(receiver Receiver) Option {
	return func(u *Updater) {
		u.receivers = append(u.receivers, receiver)
	}
}

// New creates a new updater with the required dependencies
func New(contentPath, topicFile string, reader Reader, metrics Metrics, opts ...Option) (*Updater, error) {
	u := &Updater{
		path:      contentPath,
		topicFile: topicFile,
		metrics:   metrics,
		reader:    reader,
		callbacks: []func(){},

		receivers:       []Receiver{},
		refreshInterval: 5 * time.Minute,
	}

	// apply the options
	for _, opt := range opts {
		opt(u)
	}

	// update immediately
	if err := u.Update(true); err != nil {
		return nil, err
	}
	// schedule further updates on the defined interval
	go scheduleUpdates(u.refreshInterval, func() {
		if err := u.Update(false); err != nil {
			log.Error("error when updating content", err)
		}
		for _, callback := range u.callbacks {
			callback()
		}
	})

	log.Info("updater configured to refresh content every %.0f seconds", u.refreshInterval.Seconds())

	return u, nil
}

// Update updates the content from the remote repository
func (u *Updater) Update(forceFresh bool) error {
	startTime := time.Now()
	defer u.metrics.ContentUpdated(startTime)

	if u.repo != "" {
		err := u.updateFromRemote(forceFresh)
		if err != nil {
			return err
		}
	}

	topics, articles, err := u.readFiles()
	if err != nil {
		return err
	}

	for _, receiver := range u.receivers {
		receiver(topics, articles)
	}

	return nil
}

func (u *Updater) readFiles() ([]*model.Topic, []*model.Article, error) {
	// read the main content directory to look for topic directories
	files, err := os.ReadDir(u.path)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read content directory")
	}

	newChecksums := map[string]string{}
	defer func() {
		u.fileChecksums = newChecksums
	}()

	topics := []*model.Topic{}
	articles := []*model.Article{}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		topicFilePath := filepath.Join(u.path, file.Name(), u.topicFile)
		if _, err := os.Stat(topicFilePath); os.IsNotExist(err) {
			continue
		}

		topic := u.reader.LoadTopicFromFile(topicFilePath)

		// check if the file has changed
		checksum, err := u.calculateFileChecksum(topicFilePath)
		if err != nil {
			// if we fail here, we continue without a checksum
			bugsnag.Notify(errors.Wrap(err, "failed to calculate topic checksum when updating"))
		}

		previousChecksum, ok := u.fileChecksums[topicFilePath]
		if !ok || checksum != previousChecksum {
			// there have been changes to this file
			topics = append(topics, topic)
		}

		// store the checksum for the next update
		newChecksums[topicFilePath] = checksum

		articleFiles, err := os.ReadDir(filepath.Dir(topicFilePath))
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to read topic content directory")
		}

		for _, file := range articleFiles {
			if file.IsDir() || file.Name() == filepath.Base(topicFilePath) || filepath.Ext(file.Name()) != ".md" {
				continue
			}

			articleFilepath := filepath.Join(filepath.Dir(topicFilePath), file.Name())

			// check if the file has changed
			checksum, err := u.calculateFileChecksum(articleFilepath)
			if err != nil {
				// if we fail here, we continue without a checksum
				bugsnag.Notify(errors.Wrap(err, "failed to calculate article checksum when updating"))
			}

			previousChecksum, ok := u.fileChecksums[articleFilepath]
			if !ok || checksum != previousChecksum {
				// there have been changes to this file
				article := u.reader.LoadArticleFromFile(articleFilepath, topic.Slug)
				articles = append(articles, article)
			}

			// store the checksum for the next update
			newChecksums[articleFilepath] = checksum
		}
	}

	return topics, articles, nil
}

// scheduleUpdates start a new ticker to update the content on the given interval
func scheduleUpdates(interval time.Duration, f func()) {
	for range time.Tick(interval) {
		f()
	}
}

func (u *Updater) calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
