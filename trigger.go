package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Trigger struct {
	in chan Event
}

func (t *Trigger) In() chan<- Event {
	return t.in
}

type TriggerConfig struct {
	Name  string       `yaml:"name"`
	Pipes []PipeConfig `yaml:"pipes"`
}

type PipeConfig []yaml.Node

type Config struct {
	Triggers []TriggerConfig `yaml:"triggers"`
}

func NewTrigger(ctx context.Context, config TriggerConfig) *Trigger {
	in := make(chan Event)
	t := Trigger{
		in: in,
	}

	go func() {
		defer close(in)
		wraped := inPipe()(ctx, in)

		for _, pipeConfig := range config.Pipes {
			if len(pipeConfig) == 0 {
				log.Fatal("Empty pipe config")
			}

			if pipeConfig[0].Kind != yaml.ScalarNode {
				log.Fatal("Pipe name should be a string")
			}

			name := pipeConfig[0].Value

			switch name {
			case "action_filter":
				options := pipeConfig[1]
				actions := []string{}
				for _, action := range options.Content {
					actions = append(actions, action.Value)
				}

				wraped = actionFilter(actions)(ctx, wraped)
			case "every":
				options, err := strconv.ParseInt(pipeConfig[1].Value, 10, 32)
				if err != nil {
					log.Fatal("every pipe options should be int")
				}
				wraped = everyPipe(int(options))(ctx, wraped)
			case "debounce":
				options, err := strconv.ParseInt(pipeConfig[1].Value, 10, 32)
				if err != nil {
					log.Fatal("debounce pipe options should be int")
				}
				wraped = debouncePipe(int(options))(ctx, wraped)
			case "log":
				wraped = logPipe()(ctx, wraped)
			case "webhook":
				// options, ok := pipeConfig[1].([]string)
				// if !ok {
				// 	log.Fatal("action_filter options should be array")
				// }
				wraped = logPipe()(ctx, wraped)
			case "amqp":
				fmt.Println("AMQP")
				conf := AmqpConfig{}
				err := pipeConfig[1].Decode(&conf)
				if err != nil {
					log.Fatal("amqp options should be array")
				}
				wraped = amqpPipe(conf)(ctx, wraped)
			}
		}

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
