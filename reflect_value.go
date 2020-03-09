package main

import (
	"reflect"
	"regexp"
	"strings"

	"go.starlark.net/starlark"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

type ReflectValue struct {
	val interface{}
}

func (rv ReflectValue) String() string {
	// formatted like service("name", foo="bar", ...)
	return ""
}
func (rv ReflectValue) Type() string {
	return toSnakeCase(reflect.TypeOf(rv.val).Elem().Name())
}
func (rv ReflectValue) Freeze() {

}
func (rv ReflectValue) Truth() starlark.Bool {
	return starlark.True
}
func (rv ReflectValue) Hash() (uint32, error) {
	return 0, nil
}

func (rv ReflectValue) Attr(name string) (starlark.Value, error) {
	return starlark.None, nil
}
func (rv ReflectValue) AttrNames() (out []string) {
	s := reflect.ValueOf(rv.val).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		out = append(out, toSnakeCase(typeOfT.Field(i).Name))
	}
	return
}
