package core

import (
	"context"
	"sync"
	"time"

	"github.com/jacopofilonzi/project-library/internal/config"
	"github.com/jacopofilonzi/project-library/internal/fsops"
	"github.com/jacopofilonzi/project-library/internal/platform"
	"github.com/jacopofilonzi/project-library/internal/update"
)

// ---------- update check ----------

// EventUpdate: a newer release is available (payload: update.Info).
const EventUpdate = "update:available"

// updateDelay: the startup check waits a little, so it does not compete with the first scan.
const updateDelay = 4 * time.Second

// OnUpdate is called once per new version found at startup (main sends the desktop notification).
var OnUpdate func(info update.Info, lang string)

var updates struct {
	sync.Mutex
	last *update.Info
}

// UpdateStatus returns the result of the last check (nil if none has finished yet).
func (l *Library) UpdateStatus() *update.Info {
	updates.Lock()
	defer updates.Unlock()
	return updates.last
}

// CheckUpdate checks for a new release now (Settings → About).
func (l *Library) CheckUpdate() (update.Info, error) {
	info, err := update.Check(context.Background(), Version, platform.UpdateAsset())
	if err != nil {
		return info, &fsops.Error{Code: "update", Detail: err.Error()}
	}
	updates.Lock()
	updates.last = &info
	updates.Unlock()
	if info.Available {
		emitEvent(EventUpdate, info)
	}
	return info, nil
}

// startupUpdateCheck runs the automatic check if it is enabled. Errors (offline, GitHub down) are silent.
func (l *Library) startupUpdateCheck() {
	if !l.store.Get().CheckUpdates {
		return
	}
	time.Sleep(updateDelay)
	info, err := l.CheckUpdate()
	if err != nil || !info.Available {
		return
	}
	cfg := l.store.Get()
	if cfg.NotifiedVersion == info.Latest || OnUpdate == nil {
		return
	}
	OnUpdate(info, cfg.Language)
	l.store.Update(func(c *config.Config) { c.NotifiedVersion = info.Latest })
}
