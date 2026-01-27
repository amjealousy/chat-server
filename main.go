package main

import (
	"context"
	"os"

	"wsl.test/chat"
	"wsl.test/network"
	"wsl.test/utils"
)

func main() {
	utils.IDGenerator = utils.NewIDGen()
	httpServer := network.NewServer()
	controller := chat.NewController(os.Stdout)

	httpServer.BindController(controller)
	go httpServer.Run()
	go controller.Start(context.Background())

	select {}

}
