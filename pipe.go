package main

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/streadway/amqp"
)

type pipe func(ctx context.Context, data <-chan Event) <-chan Event

func inPipe() pipe {
	return func(ctx context.Context, data <-chan Event) <-chan Event {
		return data
	}
}

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
			var counter int64 = 0
			defer close(out)

			for {
				select {
				case <-ctx.Done():
					return
				case e := <-in:
					atomic.AddInt64(&counter, 1)
					if int(counter) >= num {
						out <- e
						counter = 0
					}
				}
			}
		}()

		return out
	}
}

func debouncePipe(wait int) pipe {
	return func(ctx context.Context, in <-chan Event) <-chan Event {
		out := make(chan Event)
		go func() {
			defer close(out)

			var lastEvent Event
			waitInterval := time.Millisecond * time.Duration(wait)
			timer := time.NewTimer(waitInterval)

			for {
				select {
				case <-ctx.Done():
					return
				case lastEvent = <-in:
					timer.Reset(waitInterval)
				case <-timer.C:
					if lastEvent.Action != "" {
						out <- lastEvent
					}
				}
			}
		}()

		return out
	}
}

type AmqpConfig struct {
	DSN        string `yaml:"dsn"`
	Exchange   string `yaml:"exchange"`
	RoutingKey string `yaml:"routing_key"`
}

func amqpPipe(config AmqpConfig) pipe {
	return func(ctx context.Context, in <-chan Event) <-chan Event {
		out := make(chan Event)

		conn, err := amqp.Dial(config.DSN)
		failOnError(err, "Failed to connect to RabbitMQ")

		ch, err := conn.Channel()
		failOnError(err, "Failed to open a channel")

		go func() {
			defer close(out)
			defer conn.Close()
			defer ch.Close()

			for {
				select {
				case <-ctx.Done():
					return
				case e := <-in:
					err = ch.Publish(
						config.Exchange,   // exchange
						config.RoutingKey, // routing key
						true,              // mandatory
						false,             // immediate
						amqp.Publishing{
							ContentType: "text/plain",
							Body:        []byte(e.String()),
						})
					failOnError(err, "Failed to publish a message")
					log.Printf(" [x] Sent to %s:%s\n %s", config.Exchange, config.RoutingKey, e.String())
				}
			}
		}()

		return out
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
