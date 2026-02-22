package main

import (
	"context"
	"os"

	"wsl.test/chat"
	"wsl.test/network"
	"wsl.test/processor"
	"wsl.test/utils"
)

func main() {
	broker, b := os.LookupEnv("KafkaBroker")
	if !b {
		panic("Kafka Broker not set")
	}
	utils.IDGenerator = utils.NewIDGen()
	httpServer := network.NewServer()
	pool := processor.NewProducerPool([]string{broker}, false)
	controller := chat.NewController(os.Stdout, pool)

	httpServer.BindController(controller)
	go httpServer.Run()
	go controller.Start(context.Background())

	select {}

}
