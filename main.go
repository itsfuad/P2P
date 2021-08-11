package main

import (
	"flag"
	"log"
	"meshfile/internal/node"
	"meshfile/internal/webui"
)

var nodeInstance *node.Node

func main() {
	port := flag.Int("port", 3000, "Port to listen on")
	webUIPort := flag.Int("webui", 8080, "Web UI port")
	flag.Parse()

	config := &node.Config{
		Port:      *port,
		WebUIPort: *webUIPort,
	}

	nodeInstance = node.NewNode(config)
	webui.SetNode(nodeInstance) // Set the node instance in the webui package

	go webui.Start(*webUIPort)

	if err := nodeInstance.Start(); err != nil {
		log.Fatal(err)
	}
}
