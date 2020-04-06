package docktest

import (
	"bytes"
	"testing"

	"github.com/maxmcd/tester"
)

func TestExecInterface(te *testing.T) {
	t := tester.New(te)
	client, err := NewClient()
	t.PanicOnErr(err)
	contIPTables, err := client.ContainerCreateAndStart(ContainerCreateOptions{
		Name:  "iptales-test-container",
		Image: "nixery.dev/shell/iptables/busybox",
		Cmd:   []string{"sleep", "1000000"},
	})
	if err != nil {
		t.Error(err)
	}

	exec := ExecInterface{cont: contIPTables}

	path, err := exec.LookPath("iptables")
	t.PanicOnErr(err)
	t.Assert().Equal("/sbin/iptables", path)

	cmd := exec.Command("echo", "hi")

	var stdout bytes.Buffer
	cmd.SetStdout(&stdout)

	t.PanicOnErr(cmd.Run())
	t.Assert().Equal(stdout.Bytes(), []byte("hi\n"))
}
