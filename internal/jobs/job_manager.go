package jobs

import (
	"context"
	"errors"
	"github.com/Crowley723/conduit/internal/distributed"
	"log/slog"
	"sync"
	"time"
)

type Job interface {
	Name() string
	Run(ctx context.Context) error
	RequiresLeadership() bool
	Interval() time.Duration
}

type JobManager struct {
	jobs        []Job
	election    *distributed.Election
	logger      *slog.Logger
	wg          sync.WaitGroup
	cancelFuncs map[string]context.CancelFunc
	mu          sync.Mutex
}

func NewJobManager(election *distributed.Election, logger *slog.Logger) *JobManager {
	return &JobManager{
		jobs:        make([]Job, 0),
		election:    election,
		logger:      logger,
		cancelFuncs: make(map[string]context.CancelFunc),
	}
}

func (jm *JobManager) Register(job Job) {
	jm.jobs = append(jm.jobs, job)
}

func (jm *JobManager) Start(ctx context.Context) {
	jm.startNonLeaderJobs(ctx)

	if jm.election != nil {
		jm.wg.Add(1)
		go jm.monitorLeadership(ctx)
	} else {
		jm.startLeaderJobs(ctx)
	}
}

func (jm *JobManager) Shutdown(ctx context.Context) {
	jm.logger.Debug("Shutting down job manager...")
	jm.stopAllJobs()

	done := make(chan struct{})
	go func() {
		jm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		jm.logger.Debug("All jobs stopped cleanly")
	case <-ctx.Done():
		jm.logger.Warn("Job's failed to shutdown, exiting...")
		return
	}
}

func (jm *JobManager) monitorLeadership(ctx context.Context) {
	defer jm.wg.Done()
	ticker := time.NewTicker(jm.election.TTL / 3)
	defer ticker.Stop()

	var wasLeader bool

	for {
		select {
		case <-ctx.Done():
			jm.stopLeaderJobs()
			return
		case <-ticker.C:
			isLeader := jm.election.IsLeader()

			if isLeader && !wasLeader {
				jm.logger.Debug("Became Leader, Starting Jobs")
				jm.startLeaderJobs(ctx)
			} else if !isLeader && wasLeader {
				jm.logger.Debug("Lost Leader, Stopping Leader Jobs")
				jm.stopLeaderJobs()
			}

			wasLeader = isLeader
		}
	}
}

func (jm *JobManager) startLeaderJobs(ctx context.Context) {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	for _, job := range jm.jobs {
		if !job.RequiresLeadership() {
			continue
		}

		if _, exists := jm.cancelFuncs[job.Name()]; exists {
			continue
		}

		jobCtx, cancel := context.WithCancel(ctx)
		jm.cancelFuncs[job.Name()] = cancel

		jm.wg.Add(1)
		go func(j Job) {
			defer jm.wg.Done()
			jm.logger.Debug("Starting Job", "name", j.Name())
			if err := j.Run(jobCtx); err != nil && !errors.Is(err, context.Canceled) {
				jm.logger.Error("Job failed", "job", j.Name(), "error", err)
			}
		}(job)
	}
}

func (jm *JobManager) stopLeaderJobs() {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	for _, job := range jm.jobs {
		if !job.RequiresLeadership() {
			continue
		}

		if cancel, exists := jm.cancelFuncs[job.Name()]; exists {
			jm.logger.Debug("Stopping Job", "job", job.Name())
			cancel()
			delete(jm.cancelFuncs, job.Name())
		}
	}
}

func (jm *JobManager) startNonLeaderJobs(ctx context.Context) {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	for _, job := range jm.jobs {
		if job.RequiresLeadership() {
			continue
		}

		if _, exists := jm.cancelFuncs[job.Name()]; exists {
			continue
		}

		jobCtx, cancel := context.WithCancel(ctx)
		jm.cancelFuncs[job.Name()] = cancel

		jm.wg.Add(1)
		go func(j Job) {
			defer jm.wg.Done()
			jm.logger.Info("Starting Job", "name", j.Name())
			if err := j.Run(jobCtx); err != nil && !errors.Is(err, context.Canceled) {
				jm.logger.Error("Job failed", "job", j.Name(), "error", err)
			}
		}(job)
	}
}

func (jm *JobManager) stopAllJobs() {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	for _, job := range jm.jobs {
		if cancel, exists := jm.cancelFuncs[job.Name()]; exists {
			jm.logger.Debug("Stopping Job", "job", job.Name())
			cancel()
			delete(jm.cancelFuncs, job.Name())
		}
	}
}
