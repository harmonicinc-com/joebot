package utils

import (
	"errors"
	"strconv"
)

type PortsManager struct {
	portsInUse   map[int]bool
	allowedPorts map[int]bool

	lock chan bool
}

func NewPortsManager() *PortsManager {
	obj := new(PortsManager)
	obj.portsInUse = make(map[int](bool))
	obj.allowedPorts = make(map[int](bool))
	obj.lock = make(chan bool, 1)
	obj.lock <- true

	return obj
}

func (p *PortsManager) AddAllowedPort(port int) {
	<-p.lock
	defer func() { p.lock <- true }()

	p.allowedPorts[port] = true
}

func (p *PortsManager) ReservePort() (int, error) {
	<-p.lock
	defer func() { p.lock <- true }()

	getAllowedAndFreePort := func() (int, error) {
		if len(p.allowedPorts) == 0 {
			return 0, nil
		}

		for port := range p.allowedPorts {
			if _, isUsed := p.portsInUse[port]; !isUsed {
				_, err := GetFreePort(port, port)
				if err == nil {
					return port, nil
				}
			}
		}

		return -1, errors.New("All allowed ports are unavailable")
	}

	for i := 0; i < 10000; i++ {
		allowedAndFreePort, err := getAllowedAndFreePort()
		if err != nil {
			return 0, err
		}

		freePort, err := GetFreePort(allowedAndFreePort, allowedAndFreePort)
		if err != nil {
			return 0, err
		}
		if _, isUsed := p.portsInUse[freePort]; !isUsed {
			p.portsInUse[freePort] = true
			return freePort, nil
		}
	}

	return 0, errors.New("Unable To Reserve A Free Port")
}

func (p *PortsManager) ReleasePort(port int) error {
	<-p.lock
	defer func() { p.lock <- true }()

	if _, ok := p.portsInUse[port]; ok {
		delete(p.portsInUse, port)
		return nil
	}

	return errors.New("Unable To Release Port Because Port Not In Use: " + strconv.Itoa(port))
}
