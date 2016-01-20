package server

import (
	"net"
)

type (
	client struct {
		conn net.Conn
		collectors map[string][]string
	}

)

var (
	clients = map[string]*client{}
)

// 
func (c *client) add(collector string, tags []string) {
	if c.collectors == nil {
		c.collectors = map[string][]string{}
	}
	c.collectors[collector] = tags
}

// 
func (c *client) remove(collector string) {
	delete(c.collectors, collector)
}

func (c client) includes(collector string) bool {
	_, ok := c.collectors[collector]
	return ok
}

func (c client) tagList(collector string) []string {
	return c.collectors[collector]
}

func (c client) collectorList() []string {
	rtn := []string{}
	for key, _ := range c.collectors {
		rtn = append(rtn, key)
	}
	return rtn
}
