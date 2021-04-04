package main

import (
	"context"
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

type InputConfig struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type PipeConfig []yaml.Node

type Config struct {
	Input    InputConfig     `yaml:"input"`
	Triggers []TriggerConfig `yaml:"triggers"`
}

func NewTrigger(ctx context.Context, config TriggerConfig) *Trigger {
	in := make(chan Event)
	t := Trigger{
		in: in,
	}

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
		case "table_filter":
			options := pipeConfig[1]
			tables := []string{}
			for _, table := range options.Content {
				tables = append(tables, table.Value)
			}

			wraped = tableFilter(tables)(ctx, wraped)
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
			conf := LogConfig{}
			if len(pipeConfig) > 1 {
				pipeConfig[1].Decode(&conf)
			}
			wraped = logPipe(conf)(ctx, wraped)
		case "webhook":
			conf := WebhookConfig{}
			err := pipeConfig[1].Decode(&conf)
			if err != nil {
				log.Fatal("conf options should be array")
			}
			wraped = webhookPipe(conf)(ctx, wraped)
		case "amqp":
			conf := AmqpConfig{}
			err := pipeConfig[1].Decode(&conf)
			if err != nil {
				log.Fatal("amqp options should be array")
			}
			wraped = amqpPipe(conf)(ctx, wraped)
		}
	}

	go func() {
		defer close(in)

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
