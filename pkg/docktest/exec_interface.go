package docktest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/embly/host/pkg/exec"
	docker "github.com/fsouza/go-dockerclient"
)

var _ exec.Interface = &ExecInterface{}

type ExecInterface struct {
	cont *Container
}

func NewExecInterface(cont *Container) exec.Interface {
	return &ExecInterface{
		cont: cont,
	}
}

func (ei *ExecInterface) Command(cmd string, args ...string) exec.Cmd {
	return &CmdInterface{
		path: cmd,
		cont: ei.cont,
		args: args,
	}
}

func (ei *ExecInterface) CommandContext(ctx context.Context, cmd string, args ...string) exec.Cmd {
	return exec.New().Command(cmd, args...)
}

func (ei *ExecInterface) LookPath(file string) (string, error) {
	// This will definitely not work on all containers. for nix we specifically install
	// busybox
	stdout, _, err := ei.cont.Exec([]string{"which", "iptables"})
	if err != nil {
		return "", err
	}
	return strings.Replace(string(stdout), "\n", "", -1), nil
}

var _ exec.Cmd = &CmdInterface{}

type CmdInterface struct {
	cont   *Container
	path   string
	args   []string
	dir    string
	stdout io.Writer
	stderr io.Writer
	env    []string
}

var ErrUnimplemented = errors.New("unimplemented")

func (ci *CmdInterface) Run() error {
	dockerExec, err := client.CreateExec(docker.CreateExecOptions{
		Container:    ci.cont.ID,
		Cmd:          append([]string{ci.path}, ci.args...),
		AttachStderr: true,
		AttachStdout: true,
	})
	if err != nil {
		return err
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var output io.Writer = &stdout
	var errStream io.Writer = &stderr
	if ci.stdout != nil {
		output = io.MultiWriter(&stdout, ci.stdout)
	}
	if ci.stderr != nil {
		errStream = io.MultiWriter(&stderr, ci.stderr)
	}
	if err = client.StartExec(dockerExec.ID, docker.StartExecOptions{
		OutputStream: output,
		ErrorStream:  errStream,
	}); err != nil {
		return err
	}

	execInspect, err := client.InspectExec(dockerExec.ID)
	if err != nil {
		return err
	}
	if execInspect.ExitCode != 0 {
		return exec.CodeExitError{
			Code: execInspect.ExitCode,
		}
	}
	return nil
}

func (ci *CmdInterface) CombinedOutput() ([]byte, error) {
	var b bytes.Buffer
	ci.SetStdout(&b)
	ci.SetStderr(&b)
	err := ci.Run()
	return b.Bytes(), err
}

func (ci *CmdInterface) Output() ([]byte, error)            { return nil, ErrUnimplemented }
func (ci *CmdInterface) SetDir(dir string)                  { ci.dir = dir }
func (ci *CmdInterface) SetStdin(in io.Reader)              { panic("unimplemented") }
func (ci *CmdInterface) SetStdout(out io.Writer)            { ci.stdout = out }
func (ci *CmdInterface) SetStderr(out io.Writer)            { ci.stderr = out }
func (ci *CmdInterface) SetEnv(env []string)                { ci.env = env }
func (ci *CmdInterface) StdoutPipe() (io.ReadCloser, error) { return nil, ErrUnimplemented }
func (ci *CmdInterface) StderrPipe() (io.ReadCloser, error) { return nil, ErrUnimplemented }
func (ci *CmdInterface) Start() error                       { return ErrUnimplemented }
func (ci *CmdInterface) Wait() error                        { return ErrUnimplemented }
func (ci *CmdInterface) Stop()                              {}
