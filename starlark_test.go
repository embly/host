package host

import (
	"io/ioutil"
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
			ports=[9002, "9002/udp"],
		)
	],
)

load_balancer("all", routes={
    "localhost:8080": "counter.counter:9002",
})

`)
	file, err := RunFile(input)
	t.PanicOnErr(err)

	t.Assert().Equal(file.Containers[0].CPU, 500)
	t.Assert().Equal(file.Containers[0].Memory, 256)
	t.Assert().Equal(file.Containers[0].Ports, []string{"9002", "9002/udp"})
	t.Assert().Equal(file.Containers[0].Name, "counter")
	t.Assert().Equal(file.Containers[0].Image, "hashicorpnomad/counter-api:v1")

	t.Assert().Equal(file.Services["counter"].Name, "counter")
	t.Assert().Equal(file.Services["counter"].Count, 4)
	t.Assert().Equal(file.Services["counter"].Containers["counter"], file.Containers[0])

	t.Assert().Equal(file.LoadBalancers["all"].Routes, map[string]string{"localhost:8080": "counter.counter:9002"})
}
