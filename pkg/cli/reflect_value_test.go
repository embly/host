package cli

import (
	"testing"

	"github.com/maxmcd/tester"
)

type TestType struct {
	Foo string
	Bar string
}

func TestReflectValue(te *testing.T) {
	t := tester.New(te)

	rv := ReflectValue{val: &TestType{}}

	t.Assert().Equal(rv.AttrNames(), []string{"foo", "bar"})
	t.Assert().Equal("test_type", rv.Type())
}
