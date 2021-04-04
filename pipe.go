package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type pipe func(ctx context.Context, data <-chan Event) <-chan Event

func inPipe() pipe {
	return func(ctx context.Context, data <-chan Event) <-chan Event {
		return data
	}
}

func tableFilter(tables []string) pipe {
	tablesMap := make(map[string]bool)

	for _, table := range tables {
		tablesMap[table] = true
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
					_, ok := tablesMap[e.Table.Name]
					if ok {
						out <- e
					}
				}
			}
		}()

		return out
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

type LogConfig struct {
	Prefix string `yaml:"prefix"`
	Output string `yaml:"output"`
	Path   string `yaml:"path"`
}

func logPipe(config LogConfig) pipe {
	return func(ctx context.Context, in <-chan Event) <-chan Event {
		logger := log.New()

		switch config.Output {
		case "file":
			file, err := os.OpenFile(config.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				log.Fatalf("Failed to open log file: %v", err)
			}

			logger.SetOutput(file)
		case "stdout":
		default:
			logger.SetOutput(os.Stdout)
		}

		out := make(chan Event)
		go func() {
			defer close(out)

			for {
				select {
				case <-ctx.Done():
					return
				case e := <-in:
					logger.Info(config.Prefix, e.String())
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
			defer ch.Close()
			defer conn.Close()
			defer close(out)

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

					if err != nil {
						log.Errorf("Failed to publish a message: %s", err)
					}

					out <- e
				}
			}
		}()

		return out
	}
}

type WebhookConfig struct {
	URL    string `yaml:"url"`
	Method string `yaml:"method"`
}

func webhookPipe(config WebhookConfig) pipe {
	return func(ctx context.Context, in <-chan Event) <-chan Event {
		out := make(chan Event)

		go func() {
			tr := &http.Transport{
				MaxIdleConns:    0,
				IdleConnTimeout: 3 * time.Second,
			}
			client := &http.Client{Transport: tr}
			defer client.CloseIdleConnections()

			defer close(out)

			for {
				select {
				case <-ctx.Done():
					return
				case e := <-in:
					req, err := http.NewRequest(config.Method, config.URL, strings.NewReader(e.ToJson()))
					if err != nil {
						log.Errorf("Unable to create http request: %s", err)
					} else {
						go client.Do(req)
					}

					out <- e
				}
			}
		}()

		return out
	}
}

func failOnError(err error, msg string) {
	if err == nil {
		return
	}

	log.Fatalf("%s: %s", msg, err)
}
