package main

import (
	"context"

	"github.com/siddontang/go-mysql/canal"
)

type EventHandler struct {
	ctx      context.Context
	cancel   context.CancelFunc
	triggers []*Trigger

	canal.DummyEventHandler
}

func NewHandler(configs []TriggerConfig) *EventHandler {
	ctx, cancel := context.WithCancel(context.Background())

	eh := EventHandler{
		ctx:      ctx,
		cancel:   cancel,
		triggers: []*Trigger{},
	}

	for _, config := range configs {
		eh.triggers = append(eh.triggers, NewTrigger(ctx, config))
	}

	return &eh
}

func (h *EventHandler) Close() {
	h.cancel()
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

	for _, t := range h.triggers {
		t.In() <- event
	}

	return nil
}

func (h *EventHandler) String() string {
	return "Trigo event handler"
}
