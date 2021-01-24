package main

import "context"

type Trigger struct {
	in chan Event
}

func (t *Trigger) In() chan<- Event {
	return t.in
}

type TriggerConfig struct {
}

func NewTrigger(ctx context.Context, config TriggerConfig) *Trigger {
	in := make(chan Event)
	t := Trigger{
		in: in,
	}

	go func() {
		defer close(in)
		wraped := logPipe()(ctx, everyPipe(2)(ctx, logPipe()(ctx, in)))

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
