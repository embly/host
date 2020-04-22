package main

import (
	"log"
	"os"

	"github.com/embly/host/pkg/cli"
)

func main() {
	file, err := cli.RunFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	client, err := cli.NewAPIClient()
	if err != nil {
		panic(err)
	}

	for _, service := range file.Services {
		if err = client.DeployService(service); err != nil {
			log.Fatal(err, service)
		}
	}
}
