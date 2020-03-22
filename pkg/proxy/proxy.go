package proxy

import (
	"net"

	"github.com/sirupsen/logrus"
)

type Proxy struct {
	ip             net.IP
	cd             ConsulData
	inventory      map[string]*Service
	proxies        map[string]ProxySocket
	proxyGenerator ProxySocketGen
}

func (p *Proxy) Start() {
	updatesChan := make(chan map[string]Service)
	go p.cd.Updates(updatesChan)
	for {
		inventory := <-updatesChan
		_ = inventory
		// figure out which has updated
		// update inventory map
		// this in turn updates *Service in the Proxysocket
		p.processUpdate(inventory)
	}
}

func (p *Proxy) processUpdate(inventory map[string]Service) (new []*Service) {
	for key, service := range inventory {
		svc := service
		// check for services we don't have and add them
		if _, ok := p.inventory[key]; !ok {
			new = append(new, &svc)
			// add an entirely new service
			proxySocket, err := p.proxyGenerator.NewProxySocket(svc.protocol, p.ip, 0)
			if err != nil {
				// this will error if the input data (protocol, addr, etc..) is invalid
				// likely means we just ignore the service
				logrus.Error(err)
				continue
			}
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
	return
}
