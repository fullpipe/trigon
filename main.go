package main

import (
	"fmt"
	"log"

	"github.com/siddontang/go-mysql/canal"
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
	eh := NewHandler([]TriggerConfig{
		TriggerConfig{},
		TriggerConfig{},
	})
	defer eh.Close()

	c.SetEventHandler(eh)

	pos, err := c.GetMasterPos()
	if err != nil {
		log.Fatalln(err)
	}

	// Start canal
	c.RunFrom(pos)
}
