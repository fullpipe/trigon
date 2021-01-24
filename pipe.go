package main

import (
	"context"
	"log"
)

type pipe func(ctx context.Context, data <-chan Event) <-chan Event

func actionFilter(actions []string) pipe {
	actionsMap := make(map[string]bool)

	for _, action := range actions {
		actionsMap[action] = true
	}

	return func(ctx context.Context, data <-chan Event) <-chan Event {
		out := make(chan Event)
		go func() {
			defer close(out)

			for {
				select {
				case <-ctx.Done():
					return
				case e := <-data:
					_, ok := actionsMap[e.Action]
					if ok {
						out <- e
					}
				}
			}
		}()

		return out
	}
}

func logPipe() pipe {
	return func(ctx context.Context, in <-chan Event) <-chan Event {
		out := make(chan Event)
		go func() {
			defer close(out)

			for {
				select {
				case <-ctx.Done():
					return
				case e := <-in:
					log.Println("log pipe", e.String())
					out <- e
				}
			}
		}()

		return out
	}
}

func everyPipe(num int) pipe {
	return func(ctx context.Context, in <-chan Event) <-chan Event {
		out := make(chan Event)
		go func() {
			counter := 0
			defer close(out)

			for {
				select {
				case <-ctx.Done():
					return
				case e := <-in:
					counter++
					if counter >= num {
						out <- e
						counter = 0
					}
				}
			}
		}()

		return out
	}
}
