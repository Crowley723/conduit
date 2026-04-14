package distributed

import (
	"sync"
	"time"

	"github.com/Crowley723/conduit/internal/config"
	"github.com/Crowley723/conduit/internal/middlewares"

	"github.com/redis/go-redis/v9"
)

type Election struct {
	Redis      *redis.Client
	InstanceID string
	TTL        time.Duration
	isLeader   bool
	mu         sync.RWMutex
}

func (e *Election) IsLeader() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isLeader
}

func (e *Election) campaign(ctx middlewares.AppContext) {
	ok, err := e.Redis.SetNX(ctx, leaderKey, e.InstanceID, e.TTL).Result()
	if err != nil {
		ctx.Logger.Error("failed to campaign for leadership", "error", err, "instance", e.InstanceID)
		return
	}

	e.mu.Lock()
	wasLeader := e.isLeader

	if ok {
		e.isLeader = true
	} else {
		currentLeader, err := e.Redis.Get(ctx, leaderKey).Result()
		if err == nil && currentLeader == e.InstanceID {
			e.isLeader = true
			e.Redis.Expire(ctx, leaderKey, e.TTL)
		} else {
			e.isLeader = false
		}
	}

	e.mu.Unlock()

	if e.isLeader && !wasLeader {
		ctx.Logger.Info("became leader", "instance", e.InstanceID)
	} else if !e.isLeader && wasLeader {
		ctx.Logger.Info("lost leadership", "instance", e.InstanceID)
	}
}

func (e *Election) Start(ctx middlewares.AppContext) {
	interval := e.TTL / 3
	if interval <= 0 {
		interval = config.DefaultDistributedConfig.TTL / 3
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	e.campaign(ctx)

	for {
		select {
		case <-ctx.Done():
			e.resign(ctx)
			return
		case <-ticker.C:
			e.campaign(ctx)
		}
	}
}

func (e *Election) resign(ctx middlewares.AppContext) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isLeader {
		return
	}

	script := `
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("del", KEYS[1])
        end
        return 0
    `

	_, err := redis.NewScript(script).Run(ctx, e.Redis, []string{leaderKey}, e.InstanceID).Result()
	if err != nil {
		ctx.Logger.Error("failed to resign leadership", "error", err, "instance", e.InstanceID)
	} else {
		ctx.Logger.Info("resigned leadership", "instance", e.InstanceID)
	}

	e.isLeader = false
}
