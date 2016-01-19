package server

import (
	"net"
)

type (
	client struct {
		conn net.Conn
		tags []string
	}

)

var (
	clients = map[string]*client{}
)

// 
func (c *client) add(tag string) {
	add := true
	for i := 0; i < len(c.tags); i++ {
		if c.tags[i] == tag {
			add = false
		}
	}
	if add {
		c.tags = append(c.tags, tag)
	}
}

// 
func (c *client) remove(tag string) {
	for i := 0; i < len(c.tags); i++ {
		if c.tags[i] == tag {
			c.tags = append(c.tags[:i], c.tags[i+1:]...)
			return
		}
	}
}

func (c client) includes(tag string) bool {
	for i := 0; i < len(c.tags); i++ {
		if c.tags[i] == tag {
			return true
		}
	}
	return false
}
