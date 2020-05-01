package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/embly/host/pkg/agent"
	"github.com/embly/host/pkg/cli"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) <= 1 {
		fmt.Println("I have no name, what is my name?")
		return
	}
	command := os.Args[1]
	switch command {
	case "deploy":
		deploy()
	case "agent":
		runAgent()
	default:
		log.Fatalf("command '%s' not found", command)
	}
}

func runAgent() {
	docker0 := net.IPv4(172, 17, 0, 1)
	a, err := agent.DefaultNewAgent(docker0)
	if err != nil {
		log.Fatal("couldn't start proxy agent", err)
		return
	}
	a.Start()
}

func deploy() {
	if len(os.Args) < 3 {
		log.Fatal("This command takes one argument: <path>")
	}
	file, err := cli.RunFile(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	client, err := cli.NewAPIClient()
	if err != nil {
		log.Fatal(err)
	}

	for _, service := range file.Services {
		if err = client.DeployService(service); err != nil {
			log.Println(err)
		}
	}
}
