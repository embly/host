package main

import (
	"fmt"
	"testing"
)

func TestReflectValue(t *testing.T) {
	service := Service{}

	rv := ReflectValue{val: &service}

	fmt.Println(rv.AttrNames())
	fmt.Println(rv.Type())
}
