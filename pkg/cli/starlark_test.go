package cli

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/maxmcd/tester"
)

func tmpFileFromString(contents string) (filename string) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		panic(err)
	}
	_, _ = f.Write([]byte(contents))
	_, _ = f.Seek(0, 0)
	return f.Name()
}

func TestBasic(te *testing.T) {
	t := tester.New(te)
	input := tmpFileFromString(`
counter = service(
	"counter",
	count=4,
	containers=[
		container(
			"counter",
			image="hashicorpnomad/counter-api:v1",
			cpu=500,
			memory=256,
			connect_to=["foo", "bar"],
			environment={"FOO":"BAR"},
			ports=[9002, "9002/udp"],
		)
	]
)

load_balancer("all", routes={
    "localhost:8080": "counter.counter:9002",
})

`)
	defer os.Remove(input)
	file, err := RunFile(input)
	t.PanicOnErr(err)

	t.Assert().Equal(file.Containers[0].CPU, 500)
	t.Assert().Equal(file.Containers[0].Memory, 256)
	t.Assert().Equal(file.Containers[0].Ports, []string{"9002", "9002/udp"})
	t.Assert().Equal(file.Containers[0].Name, "counter")
	t.Assert().Equal(file.Containers[0].Image, "hashicorpnomad/counter-api:v1")
	t.Assert().Equal(file.Containers[0].ConnectTo, []string{"foo", "bar"})
	t.Assert().Equal(file.Containers[0].Environment, map[string]string{"FOO": "BAR"})

	t.Assert().Equal(file.Services["counter"].Name, "counter")
	t.Assert().Equal(file.Services["counter"].Count, 4)
	t.Assert().Equal(file.Services["counter"].Containers["counter"], file.Containers[0])

	t.Assert().Equal(file.LoadBalancers["all"].Routes, map[string]string{"localhost:8080": "counter.counter:9002"})
}

func TestErrorCases(te *testing.T) {
	t := tester.New(te)
	for source, errorContains := range map[string]string{
		`container("c", image="")`:                         "must have an image",
		`container("c", ports=["f9002"], image="asdf")`:    "ports must be formatted",
		`container("c", ports=[12.0], image="asdf")`:       "got type: float",
		`container("c", ports=["0"], image="asdf")`:        "can't be zero",
		`container("c", ports=["70000"], image="asdf")`:    "too high",
		`container("c", ports="", image="asdf")`:           "list of ports",
		`container("c", connect_to="", image="asdf")`:      "container() parameter 'connect_to'",
		`container("c", connect_to=[1, ""], image="asdf")`: "onnect_to expects a list of strings",
		`container("c", foo="bar", image="asdf")`:          "unexpected keyword argument",
		`load_balancer("all", routes=5)`:                   "parameter 'routes'",
		`load_balancer("all", routes={1:2})`:               "dictionary of strings",
		`load_balancer("all", routes={"1":2})`:             "dictionary of strings",
		`load_balancer("all", foo="bar")`:                  "unexpected keyword argument",
		`service("hi", containers=4)`:                      `unexpected type 'int'`,
		`service("hi", containers=[4])`:                    `got a value '[4]'`,
		`service("all", foo="bar")`:                        "unexpected keyword argument",
		`service("all");service("all")`:                    "names must be unique",
	} {
		input := tmpFileFromString(source)
		defer os.Remove(input)
		_, err := RunFile(input)
		t.ErrorContains(err, errorContains, input)
	}
}
