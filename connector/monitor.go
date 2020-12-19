package main

import (
	"log"
)

func main() {
	c, err := newConnector()
	if err != nil {
		log.Fatalf("socket: %v\n", err)
	}
	defer c.Close()

	if err := c.Connect(); err != nil {
		log.Fatalf("bind: %v\n", err)
	}

	if err := c.enableMonitor(true); err != nil {
		log.Fatalf("start monitoring: %v\n", err)
	}
	defer c.enableMonitor(false)

	log.Println("monitoring...")
	for {
		c.readEvent()
	}
}
