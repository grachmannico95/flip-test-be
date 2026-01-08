package eventbus

import (
	"context"
	"sync"

	"github.com/grachmannico95/flip-test-be/pkg/logger"
	"github.com/grachmannico95/flip-test-be/pkg/retry"
)

type EventBus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(eventType EventType, consumer Consumer) error
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

type eventBus struct {
	channels      map[EventType]chan Event
	consumers     map[EventType][]Consumer
	mu            sync.RWMutex
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	logger        *logger.Logger
	channelBuffer int
	started       bool
}

type Config struct {
	ChannelBuffer int
	MaxRetries    int
}

func New(log *logger.Logger, cfg *Config) EventBus {
	if cfg == nil {
		cfg = &Config{
			ChannelBuffer: 1000,
			MaxRetries:    5,
		}
	}

	return &eventBus{
		channels:      make(map[EventType]chan Event),
		consumers:     make(map[EventType][]Consumer),
		logger:        log,
		channelBuffer: cfg.ChannelBuffer,
	}
}

func (eb *eventBus) Subscribe(eventType EventType, consumer Consumer) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if _, exists := eb.channels[eventType]; !exists {
		eb.channels[eventType] = make(chan Event, eb.channelBuffer)
	}

	eb.consumers[eventType] = append(eb.consumers[eventType], consumer)

	return nil
}

func (eb *eventBus) Start(ctx context.Context) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.started {
		return nil
	}

	eb.ctx, eb.cancel = context.WithCancel(ctx)

	for eventType, consumers := range eb.consumers {
		ch := eb.channels[eventType]

		for _, consumer := range consumers {
			workerCount := consumer.GetWorkerCount()
			eb.logger.Info(eb.ctx, "Starting workers",
				"event_type", eventType,
				"worker_count", workerCount,
			)

			for i := 0; i < workerCount; i++ {
				eb.wg.Add(1)
				go eb.worker(eb.ctx, ch, consumer, i)
			}
		}
	}

	eb.started = true
	eb.logger.Info(eb.ctx, "Event bus started")

	return nil
}

func (eb *eventBus) worker(ctx context.Context, ch <-chan Event, consumer Consumer, workerID int) {
	defer eb.wg.Done()

	eb.logger.Debug(ctx, "Worker started", "worker_id", workerID)

	for {
		select {
		case <-ctx.Done():
			eb.logger.Debug(ctx, "Worker stopping", "worker_id", workerID)
			return
		case event, ok := <-ch:
			if !ok {
				eb.logger.Debug(ctx, "Channel closed, worker stopping", "worker_id", workerID)
				return
			}

			eb.processEvent(ctx, event, consumer, workerID)
		}
	}
}

func (eb *eventBus) processEvent(ctx context.Context, event Event, consumer Consumer, workerID int) {
	// Create context with event ID for tracing
	eventCtx := ctx
	if event.ID != "" {
		eventCtx = logger.WithTraceID(ctx, event.ID)
	}

	eb.logger.Debug(eventCtx, "Processing event",
		"event_id", event.ID,
		"event_type", event.Type,
		"worker_id", workerID,
	)

	// Retry with exponential backoff
	err := retry.Do(eventCtx, func() error {
		return consumer.Consume(eventCtx, event)
	}, retry.WithMaxAttempts(5))

	if err != nil {
		eb.logger.Error(eventCtx, "Failed to process event after retries",
			"event_id", event.ID,
			"event_type", event.Type,
			"worker_id", workerID,
			"error", err,
		)
	} else {
		eb.logger.Debug(eventCtx, "Event processed successfully",
			"event_id", event.ID,
			"event_type", event.Type,
			"worker_id", workerID,
		)
	}
}

func (eb *eventBus) Publish(ctx context.Context, event Event) error {
	eb.mu.RLock()
	ch, exists := eb.channels[event.Type]
	eb.mu.RUnlock()

	if !exists {
		eb.logger.Warn(ctx, "No channel for event type",
			"event_type", event.Type,
			"event_id", event.ID,
		)
		return nil
	}

	// Non-blocking send
	select {
	case ch <- event:
		eb.logger.Debug(ctx, "Event published",
			"event_type", event.Type,
			"event_id", event.ID,
		)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Channel is full, log warning but don't block
		eb.logger.Warn(ctx, "Event channel full, event dropped",
			"event_type", event.Type,
			"event_id", event.ID,
		)
		return nil
	}
}

func (eb *eventBus) Shutdown(ctx context.Context) error {
	eb.logger.Info(ctx, "Shutting down event bus")

	// Signal all workers to stop
	if eb.cancel != nil {
		eb.cancel()
	}

	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		eb.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		eb.logger.Info(ctx, "Event bus shutdown complete")
		return nil
	case <-ctx.Done():
		eb.logger.Warn(ctx, "Event bus shutdown timeout")
		return ctx.Err()
	}
}
