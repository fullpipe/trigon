package main

import "context"

type Trigger struct {
	in <-chan Event
}

type TriggerConfig struct {
}

func NewTrigger(ctx context.Context, in <-chan Event, config TriggerConfig) *Trigger {
	t := Trigger{
		in: in,
	}

	go func() {
		wraped := logPipe()(ctx, everyPipe(2)(ctx, in))

		for {
			select {
			case <-ctx.Done():
				return
			case <-wraped:
			}
		}
	}()

	return &t
}
