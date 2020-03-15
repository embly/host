package main

import (
	"fmt"
	"log"
	"os"

	"github.com/embly/host"
)

func main() {
	file, err := host.RunFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	for _, service := range file.Services {
		job := host.ServiceToJob(service)
		err := host.DeployIsh(job)
		fmt.Println(err)
	}

}
