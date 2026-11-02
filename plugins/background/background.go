// Package background runs an app's Go code while it isn't open, as the phone
// finds the time: syncing a feed overnight, uploading what waited for a network.
// BGTaskScheduler on iOS, WorkManager on Android; elsewhere Schedule reports
// errors.ErrUnsupported.
//
// The system starts the app's process to run a task, without showing it, and
// main runs as ever: call Handle before App.Run, so that the handler is there
// by then.
//
//	background.Handle("sync", func(ctx context.Context) error {
//		return syncFeed(ctx)
//	})
//	mygo.App.WhenReady(func() {
//		go background.Schedule(context.Background(), "sync", background.Options{
//			Every: 4 * time.Hour, Network: true,
//		})
//	})
//
// When a task runs is the system's choice: after Delay, around Every, with what
// it knows of the battery and of how much the user opens the app. iOS gives a
// task about 30 seconds, or minutes with Long; Android about 10 minutes. A
// handler stops when its context ends. Tasks outlive the process and reboots
// until Cancel.
package background

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/egoist/mygo"
)

const pluginID = "background"

// Options say when a task may run.
type Options struct {
	// Every repeats it, at least 15 minutes apart on either system. Without it
	// the task runs once.
	Every time.Duration
	// Delay is the least it waits before it runs the first time.
	Delay time.Duration
	// Network waits for a network.
	Network bool
	// Charging waits for the phone to charge.
	Charging bool
	// Long asks for minutes, which on iOS come rarely, overnight and charging,
	// rather than half a minute.
	Long bool
}

var tasks struct {
	sync.Mutex
	handlers map[string]func(context.Context) error
	running  map[int64]context.CancelFunc
	wired    bool
}

// Handle says what a task does, before App.Run. It runs on a goroutine of its
// own, off the main thread. A task whose handler returns an error runs again
// later, on Android.
func Handle(name string, fn func(ctx context.Context) error) {
	tasks.Lock()
	defer tasks.Unlock()
	if tasks.handlers == nil {
		tasks.handlers = map[string]func(context.Context) error{}
		tasks.running = map[int64]context.CancelFunc{}
	}
	tasks.handlers[name] = fn
	if tasks.wired {
		return
	}
	tasks.wired = true
	p := mygo.Native(pluginID)
	p.On("run", func(data []byte) {
		var e runEvent
		if json.Unmarshal(data, &e) == nil {
			run(e, finish)
		}
	})
	p.On("stop", func(data []byte) {
		var e runEvent
		if json.Unmarshal(data, &e) == nil {
			stop(e.Run)
		}
	})
	mygo.App.WhenReady(func() {
		// What the system started the process for waited for these handlers.
		go func() {
			if err := p.Call(context.Background(), "ready", nil, nil); err != nil && p.Available() {
				log.Printf("background: %v", err)
			}
		}()
	})
}

// Schedule asks the system to run a task, in place of one of its name. Call it
// off the main goroutine, or with go.
func Schedule(ctx context.Context, name string, o Options) error {
	return mygo.Native(pluginID).Call(ctx, "schedule", map[string]any{
		"name": name, "every": o.Every.Seconds(), "delay": o.Delay.Seconds(),
		"network": o.Network, "charging": o.Charging, "long": o.Long,
	}, nil)
}

// Cancel takes a task off the system's list.
func Cancel(ctx context.Context, name string) error {
	return mygo.Native(pluginID).Call(ctx, "cancel", map[string]any{"name": name}, nil)
}

type runEvent struct {
	Run  int64  `json:"run"`
	Name string `json:"name"`
}

// run runs a task's handler, and tells what came of it. What has no handler
// finishes: the app no longer does it.
func run(e runEvent, done func(run int64, ok bool)) {
	tasks.Lock()
	fn := tasks.handlers[e.Name]
	if fn == nil {
		tasks.Unlock()
		done(e.Run, true)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	tasks.running[e.Run] = cancel
	tasks.Unlock()
	go func() {
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("background: task %s: %v", e.Name, r)
					err = context.Canceled
				}
			}()
			return fn(ctx)
		}()
		if err != nil {
			log.Printf("background: task %s: %v", e.Name, err)
		}
		tasks.Lock()
		delete(tasks.running, e.Run)
		tasks.Unlock()
		cancel()
		done(e.Run, err == nil)
	}()
}

// stop ends a task's context, as the system's time for it ended.
func stop(id int64) {
	tasks.Lock()
	cancel := tasks.running[id]
	tasks.Unlock()
	if cancel != nil {
		cancel()
	}
}

func finish(id int64, ok bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := mygo.Native(pluginID).Call(ctx, "done", map[string]any{"run": id, "ok": ok}, nil); err != nil {
		log.Printf("background: %v", err)
	}
}
