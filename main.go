package main

import (
	"flag"
	"io/ioutil"

	"github.com/siddontang/go-mysql/canal"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func main() {
	configPath := flag.String("config", "trigon.yaml", "Path to config file")
	flag.Parse()

	config := getTriggersConfig(configPath)

	cfg := canal.NewDefaultConfig()
	cfg.Addr = config.Input.Host
	cfg.User = config.Input.User
	cfg.Password = config.Input.Password
	cfg.Dump.ExecutionPath = ""

	c, err := canal.NewCanal(cfg)
	if err != nil {
		log.Fatalln(err)
	}

	eh := NewHandler(config)
	defer eh.Close()

	c.SetEventHandler(eh)

	pos, err := c.GetMasterPos()
	if err != nil {
		log.Fatalln(err)
	}

	// Start canal
	log.Info("Starting from binlog position:", pos)
	c.RunFrom(pos)
}

func getTriggersConfig(configPath *string) Config {
	yamlFile, err := ioutil.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("yamlFile.Get err #%v ", err)
	}

	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return config
}
