package eventbus

import "context"

type Consumer interface {
	Consume(ctx context.Context, event Event) error
	GetWorkerCount() int
}
