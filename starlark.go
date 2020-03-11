package host

import (
	"fmt"

	"github.com/pkg/errors"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

func NewProject() *Project {
	return &Project{
		Services:      map[string]Service{},
		LoadBalancers: map[string]LoadBalancer{},
	}
}

// Project is functioning as some kind of registry of data
// access patterns here are likely going to have to be very flexible
// we'll also need for it to work accross many file imports
// very subject to change
type Project struct {
	Services      map[string]Service
	LoadBalancers map[string]LoadBalancer
	Containers    map[string]Container
}

// NewFile creates a new file
func NewFile() *File {
	return &File{
		Services:      map[string]Service{},
		LoadBalancers: map[string]LoadBalancer{},
	}
}

// File lets us ignore the project problems above and just keep it scoped to
// a file
type File struct {
	Services      map[string]Service
	LoadBalancers map[string]LoadBalancer
	Containers    []Container
}

type Container struct {
	id          int
	Name        string
	Image       string
	CPU         int
	Memory      int
	Command     string
	Ports       []string
	ports       []Port
	ConnectTo   []string
	Environment map[string]string
}

type Port struct {
	isUDP  bool
	number int
}

type LoadBalancer struct {
	Name   string
	Routes map[string]string
}

type Service struct {
	Name       string
	Count      int
	Containers map[string]Container
}

func listOfPortsToInt(val starlark.Value) (out []string, err error) {
	list, ok := val.(*starlark.List)
	if !ok {
		err = errors.Errorf("expected a list, but got: %s", val.Type())
		return
	}
	l := list.Len()
	for i := 0; i < l; i++ {
		item := list.Index(i)
		switch item.Type() {
		case "int":
			out = append(out, item.String())
		case "string":
			out = append(out, item.(starlark.String).GoString())
		default:
			err = errors.Errorf("ports can be int or string, but got type: %s", item.Type())
			return
		}
	}
	return
}

func (p *File) Validate() (err error) {
	validAddressableDomains := map[string]struct{}{}
	for sName, service := range p.Services {
		for cName, container := range service.Containers {
			for _, value := range container.Ports {
				// TODO parse ports
				validAddressableDomains[cName+"."+sName+":"+value] = struct{}{}
			}
		}
	}

	for _, lb := range p.LoadBalancers {
		for host, dest := range lb.Routes {
			if _, ok := validAddressableDomains[dest]; !ok {
				_ = host
				err = errors.Errorf("%s is not a valid service domain", dest)
				return
			}
		}
	}
	return
}

func (p *File) Container(thread *starlark.Thread, fn *starlark.Builtin,
	args starlark.Tuple, kwargs []starlark.Tuple) (v starlark.Value, err error) {
	name := string(args.Index(0).(starlark.String))
	container := Container{}
	container.Name = name

	for _, kwarg := range kwargs {
		key := string(kwarg.Index(0).(starlark.String))
		switch key {
		case "image":
			container.Image = string(kwarg.Index(1).(starlark.String))
		case "cpu":
			i, _ := kwarg.Index(1).(starlark.Int).Int64()
			container.CPU = int(i)
		case "ports":
			container.Ports, err = listOfPortsToInt(kwarg.Index(1))
			if err != nil {
				return
			}
		case "memory":
			i, _ := kwarg.Index(1).(starlark.Int).Int64()
			container.Memory = int(i)
		default:
			err = errors.Errorf(`container() got an unexpected keyword argument '%s'"`, key)
			return
		}
	}

	// for later lookups
	container.id = len(p.Containers)

	p.Containers = append(p.Containers, container)
	return &ReflectValue{val: &container}, nil
}

func (p *File) Service(thread *starlark.Thread, fn *starlark.Builtin,
	args starlark.Tuple, kwargs []starlark.Tuple) (v starlark.Value, err error) {
	name := string(args.Index(0).(starlark.String))
	service := Service{Containers: map[string]Container{}}
	service.Name = name
	for _, kwarg := range kwargs {
		key := string(kwarg.Index(0).(starlark.String))
		value := kwarg.Index(1)
		switch key {
		case "count":
			i, _ := value.(starlark.Int).Int64()
			service.Count = int(i)
		case "containers":
			list, ok := value.(*starlark.List)
			if !ok {
				err = errors.Errorf(`service() argument 'containers' takes a list, but got an unexpected type '%s'"`, value.Type())
				return
			}
			for i := 0; i < list.Len(); i++ {
				val := list.Index(i)
				maybeErr := errors.Errorf(`service() argument 'containers' takes a list of containers, got a value '%s' instead"`, value.String())
				if val.Type() != "container" {
					err = maybeErr
					return
				}
				rv, ok := val.(*ReflectValue)
				if !ok {
					err = maybeErr
					return
				}
				cont, ok := rv.val.(*Container)
				if !ok {
					err = maybeErr
					return
				}
				if _, ok := service.Containers[cont.Name]; ok {
					err = errors.Errorf(`this service already has a container named '%s'. service container names must be unique`, cont.Name)
					return
				}
				service.Containers[cont.Name] = *cont
			}
		default:
			err = errors.Errorf(`service() got an unexpected keyword argument '%s'"`, key)
			return
		}
	}
	if _, ok := p.Services[name]; ok {
		err = errors.Errorf(`a service was already created with name '%s'. service names must be unique`, name)
		return
	}
	p.Services[name] = service
	return &ReflectValue{&service}, nil
}

func (file *File) LoadBalancer(thread *starlark.Thread, fn *starlark.Builtin,
	args starlark.Tuple, kwargs []starlark.Tuple) (v starlark.Value, err error) {

	name := string(args.Index(0).(starlark.String))
	lb := LoadBalancer{Routes: map[string]string{}}
	lb.Name = name
	for _, kwarg := range kwargs {
		key := string(kwarg.Index(0).(starlark.String))
		val := kwarg.Index(1)
		switch key {
		case "routes":
			maybeErr := errors.Errorf("load_balancer() parameter 'routes' expects type 'dict' instead got '%s'", val.String())
			if val.Type() != "dict" {
				err = maybeErr
				return
			}
			dict, ok := val.(starlark.IterableMapping)
			if !ok {
				err = maybeErr
				return
			}
			items := dict.Items()
			for _, item := range items {
				key := item.Index(0)
				value := item.Index(1)
				ks, ok := key.(starlark.String)
				if !ok {
					err = errors.Errorf("load_balancer() routes expects a dictionary of strings, but got value %s", key.String())
					return
				}
				vs, ok := value.(starlark.String)
				if !ok {
					err = errors.Errorf("load_balancer() routes expects a dictionary of strings, but got value %s", value.String())
					return
				}
				lb.Routes[ks.GoString()] = vs.GoString()
			}
		default:
			err = errors.Errorf(`load_balancer() got an unexpected keyword argument '%s'"`, key)
			return
		}
	}
	file.LoadBalancers[name] = lb
	return &ReflectValue{&lb}, nil
}

func RunFile(filename string) (file *File, err error) {
	file = NewFile()
	resolve.AllowFloat = true
	resolve.AllowSet = true
	resolve.AllowLambda = true
	// Execute Starlark program in a file.
	thread := &starlark.Thread{Name: ""}
	_, err = starlark.ExecFile(thread, filename, nil, starlark.StringDict{
		"service":       starlark.NewBuiltin("service", file.Service),
		"load_balancer": starlark.NewBuiltin("load_balancer", file.LoadBalancer),
		"container":     starlark.NewBuiltin("container", file.Container),
	})
	if er, ok := err.(*starlark.EvalError); ok {
		fmt.Println(er.Backtrace())
	}
	err = file.Validate()
	return
}
