package scheduler

import (
	"context"
	"fmt"
	"time"
)

type Scheduler struct {
	timeToExecute           time.Time
	function                func()
	currentContextCanceller context.CancelFunc
}

func New(timeToExecute time.Time, f func()) *Scheduler {
	s := &Scheduler{
		timeToExecute: timeToExecute,
		function:      f,
	}

	s.reschedule()

	return s
}

func (s *Scheduler) Shutdown() {
	s.cancelCurrentContext()
}

func (s *Scheduler) cancelCurrentContext() {
	if s.currentContextCanceller != nil {
		s.currentContextCanceller()
		s.currentContextCanceller = nil
	}
}

func (s *Scheduler) calculateNextTime() time.Time {
	now := time.Now()

	nextTime := time.Date(now.Year(), now.Month(), now.Day(), s.timeToExecute.Hour(), s.timeToExecute.Minute(), s.timeToExecute.Second(), 0, now.Location())
	if nextTime.Before(now) {
		nextTime = nextTime.Add(24 * time.Hour)
	}

	return nextTime
}

func (s *Scheduler) reschedule() {
	log("Rescheduling...")
	s.cancelCurrentContext()

	ctx, cancel := context.WithCancel(context.Background())
	s.currentContextCanceller = cancel

	nextTime := s.calculateNextTime()
	go s.runAtTime(ctx, nextTime)
}

func (s *Scheduler) runAtTime(ctx context.Context, t time.Time) {
	log("task scheduled for %v", t)

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(t)):
			// run the function and reschedule
			s.function()
			s.reschedule()
		}
	}
}

func log(format string, a ...any) {
	fmt.Printf("[scheduler] %s - %s\n", time.Now().Format("2006/01/02 15:04:05"), fmt.Sprintf(format, a...))
}
