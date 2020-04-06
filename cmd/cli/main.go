package main

import (
	"fmt"
	"log"
	"os"

	"github.com/embly/host/pkg/cli"
)

func main() {
	file, err := cli.RunFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	for _, service := range file.Services {
		job := cli.ServiceToJob(service)
		err := cli.DeployIsh(job)
		fmt.Println(err)
	}
}
