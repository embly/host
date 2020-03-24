package agent

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Proxy struct {
	ip             net.IP
	cd             ConsulData
	lock           sync.RWMutex
	inventory      map[string]*Service
	proxies        map[string]ProxySocket
	proxyGenerator ProxySocketGen

	cond sync.Cond
}

func NewProxy() *Proxy {
	var cd ConsulData
	var err error
	for _, sleepFor := range EndpointDialTimeouts {
		cd, err = NewConsulData()
		if err != nil {
			time.Sleep(sleepFor)
		}
	}
	if err != nil {
		panic(err)
	}
	fmt.Println("connected")
	return &Proxy{
		ip:             net.IPv4(127, 0, 0, 1),
		cd:             cd,
		proxyGenerator: DefaultProxySocketGen,
		cond:           sync.Cond{L: &sync.Mutex{}},
	}
}

func (p *Proxy) Start() {
	if p.inventory == nil {
		p.inventory = map[string]*Service{}
	}
	if p.proxies == nil {
		p.proxies = map[string]ProxySocket{}
	}
	updatesChan := make(chan map[string]Service)
	logrus.Debug("started listening for updates")
	go p.cd.Updates(updatesChan)
	for {
		inventory := <-updatesChan
		logrus.Debug("got inventory update")
		_ = inventory
		// figure out which has updated
		// update inventory map
		// this in turn updates *Service in the Proxysocket
		p.processUpdate(inventory)
		p.cond.Broadcast()
	}
}

// wait is used for testing to wait for the next inventory update
func (p *Proxy) wait() {
	p.cond.L.Lock()
	p.cond.Wait()
	p.cond.L.Unlock()
}

func (p *Proxy) processUpdate(inventory map[string]Service) (new []*Service) {
	p.lock.Lock()
	for key, service := range inventory {
		// check for services we don't have and add them
		_, ok := p.inventory[key]
		if !ok {
			logrus.Info("adding proxy for ", key)
			svc := service
			new = append(new, &svc)
			// add an entirely new service
			proxySocket, err := p.proxyGenerator.NewProxySocket(svc.protocol, p.ip, 0)
			if err != nil {
				// this will error if the input data (protocol, addr, etc..) is invalid
				// likely means we just ignore the service
				logrus.Error(err)
				continue
			}
			logrus.Info(proxySocket, err)
			go proxySocket.ProxyLoop(&svc)
			p.inventory[key] = &svc
			p.proxies[key] = proxySocket
			// TODO: add new proxy here as well
		} else {
			// update the task list on an existing serivice
			p.inventory[key].processUpdate(service.inventory)
		}
	}

	// check for services we no longer have and shut them down
	for key := range p.inventory {
		if _, ok := inventory[key]; !ok {
			// marking it as dead will stop new requests
			// drop the proxy from the map and it will eventually exit
			p.inventory[key].alive = false
			p.proxies[key].Close()
			delete(p.inventory, key)
			delete(p.proxies, key)
		}
	}
	p.lock.Unlock()
	return
}
