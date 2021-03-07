package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/siddontang/go-mysql/canal"
	"gopkg.in/yaml.v3"
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
	fmt.Println(GetTriggersConfig())
	eh := NewHandler(GetTriggersConfig())
	defer eh.Close()

	c.SetEventHandler(eh)

	pos, err := c.GetMasterPos()
	if err != nil {
		log.Fatalln(err)
	}

	// Start canal
	c.RunFrom(pos)
}

func GetTriggersConfig() Config {
	yamlFile, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return config
}
