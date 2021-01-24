package main

import (
	"context"
	"fmt"
	"log"

	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/schema"
	"github.com/tidwall/sjson"
)

func main() {
	cfg := canal.NewDefaultConfig()
	cfg.Addr = "127.0.0.1:3320"
	cfg.User = "root"
	cfg.Password = "root"

	cfg.Dump.ExecutionPath = ""

	fmt.Println(cfg)
	c, err := canal.NewCanal(cfg)
	if err != nil {
		log.Fatalln(err)
	}

	// Register a handler to handle RowsEvent
	eh := NewHandler([]string{"update", "delete"})
	defer eh.Close()

	c.SetEventHandler(eh)

	pos, err := c.GetMasterPos()
	if err != nil {
		log.Fatalln(err)
	}

	// Start canal
	c.RunFrom(pos)
}

type EventHandler struct {
	actions map[string]bool
	ctx     context.Context
	cancel  context.CancelFunc
	canal.DummyEventHandler
	pipe     pipe
	in       chan Event
	wraped   <-chan Event
	triggers []*Trigger
}

func NewHandler(actions []string) *EventHandler {
	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan Event)

	eh := EventHandler{
		actions:  make(map[string]bool),
		ctx:      ctx,
		cancel:   cancel,
		in:       in,
		triggers: []*Trigger{},
	}

	eh.triggers = append(eh.triggers, NewTrigger(ctx, in, TriggerConfig{}))

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

	for _, action := range actions {
		eh.actions[action] = true
	}

	return &eh
}

func (h *EventHandler) Close() {
	h.cancel()
	close(h.in)
}

func (h *EventHandler) OnRow(e *canal.RowsEvent) error {
	event := Event{
		Action: e.Action,
		Table:  e.Table,
	}

	switch e.Action {
	case canal.InsertAction:
		event.After = e.Rows[0]
	case canal.DeleteAction:
		event.Before = e.Rows[0]
	case canal.UpdateAction:
		for i := 0; i < len(e.Rows); i += 2 {
			event.Before = e.Rows[i]
			event.After = e.Rows[i+1]
		}
	}

	h.in <- event

	_, ok := h.actions[e.Action]
	if !ok {
		return nil
	}

	var json string
	var err error

	switch e.Action {
	case canal.InsertAction:
	case canal.DeleteAction:
		json, err = toJson(e.Table, e.Rows[0])
		if err != nil {
			log.Fatal(err)
		}
	case canal.UpdateAction:
		json, err = toJson(e.Table, e.Rows[1])
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println(e.Action, e.String(), e.Table.Schema, e.Header, json)
	return nil
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

type pipe func(ctx context.Context, data <-chan Event) <-chan Event

type Event struct {
	Table  *schema.Table
	Action string
	Before []interface{}
	After  []interface{}
}

func (e *Event) String() string {
	return e.ToJson()
}

func (e *Event) ToJson() string {
	json := `{"before": null, "after": null}`
	json, _ = sjson.Set(json, "action", e.Action)
	json, _ = sjson.Set(json, "table", e.Table.Name)

	var before, after string

	before, _ = toJson(e.Table, e.Before)
	if before != "" {
		json, _ = sjson.SetRaw(json, "before", before)
	}

	after, _ = toJson(e.Table, e.After)
	if after != "" {
		json, _ = sjson.SetRaw(json, "after", after)
	}

	return json
}

func toJson(table *schema.Table, row []interface{}) (string, error) {
	json := "{}"

	for _, col := range table.Columns {
		val, err := table.GetColumnValue(col.Name, row)
		if err != nil {
			return "", err
		}

		json, _ = sjson.Set(json, col.Name, val)
	}

	return json, nil
}

func (h *EventHandler) String() string {
	return "Trigo event handler"
}
