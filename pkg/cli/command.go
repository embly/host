package cli

import (
	"context"
	"log"
	"net"
	"os"

	"github.com/embly/host/pkg/agent"
	"github.com/fatih/color"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

/*
type Command interface {
	// Help should return long-form help text that includes the command-line
	// usage, a brief few sentences explaining the function of the command,
	// and the complete list of flags the command accepts.
	Help() string

	// Run should run the actual command with the given CLI instance and
	// command-line arguments. It should return the exit status when it is
	// finished.
	//
	// There are a handful of special exit codes this can return documented
	// above that change behavior.
	Run(args []string) int

	// Synopsis should return a one-line, short synopsis of the command.
	// This should be less than 50 characters ideally.
	Synopsis() string
}
*/

type Command struct {
	help     string
	synopsis string
	run      func([]string) error
}

var _ cli.Command = &Command{}

func (c *Command) Help() string     { return c.help }
func (c *Command) Synopsis() string { return c.synopsis }
func (c *Command) Run(args []string) int {
	if err := c.run(args); err != nil {
		logrus.Error(err)
		return 1
	}
	return 0
}

func RunCommand(version string) {
	c := cli.NewCLI("twelve", version)
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"launch": func() (cli.Command, error) {
			return &Command{
				help:     "hi",
				synopsis: "Launches services for local dev",
				run:      RunStart,
			}, nil
		},
		"stop": func() (cli.Command, error) {
			return &Command{
				help:     "hi",
				synopsis: "Stops services for local development",
				run:      RunStop,
			}, nil
		},
		"run": func() (cli.Command, error) {
			return &Command{
				help:     "hi",
				synopsis: "Runs a service",
				run:      RunRun,
			}, nil
		},
		"status": func() (cli.Command, error) {
			return &Command{
				help:     "hi",
				synopsis: "Display service statuses",
				run:      func(a []string) error { return nil },
			}, nil
		},
		"agent": func() (cli.Command, error) {
			return &Command{
				help:     "hi",
				synopsis: "Runs an embly agent",
				run:      RunAgent,
			}, nil
		},
	}
	// TODO
	// c.HelpFunc = func(in map[string]cli.CommandFactory) string {}
	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}
	os.Exit(exitStatus)
}

func RunAgent(args []string) error {
	docker0 := net.IPv4(172, 17, 0, 1)
	a, err := agent.DefaultNewAgent(docker0)
	if err != nil {
		log.Fatal("couldn't start proxy agent", err)
		return err
	}
	a.Start()
	return nil
}

func RunRun(args []string) error {
	if len(args) < 1 {
		return errors.New("this command takes one argument: <path>")
	}
	file, err := RunFile(args[0])
	if err != nil {
		return err
	}
	client, err := NewAPIClient()
	if err != nil {
		return errors.Wrap(err, "error creating new client")
	}
	healthy, err := client.Healthy()
	if !healthy {
		color.New(color.FgRed, color.Bold).Print("\nCouldn't connect to the local embly client. Is it running? Run with \"embly start\"\n\n")
		return err
	}
	resp, err := client.grpcClient.Health(context.Background(), nil)
	if err != nil {
		return errors.Wrap(err, "error fetching client health")
	}
	_ = resp
	for _, service := range file.Services {
		if err = client.DeployService(service); err != nil {
			log.Println(err)
		}
	}
	return nil
}

func RunStart(args []string) error {
	client, err := NewAPIClient()
	if err != nil {
		return errors.Wrap(err, "error creating new client")
	}

	return client.StartLocalServices()
}

func RunStop(args []string) error {
	client, err := NewAPIClient()
	if err != nil {
		return errors.Wrap(err, "error creating new client")
	}

	return client.StopLocalServices()
}
