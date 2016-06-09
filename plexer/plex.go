// Package plexer provides the means to live publish or store stats. In pulse,
// influx is a BatchPublisher, and mist is a SinglePublisher.
package plexer

import (
	"errors"

	"github.com/jcelliott/lumber"
)

var (
	MissingPublisher = errors.New("A publisher is needed")
)

type (
	BatchPublisher  func(MessageSet) error
	SinglePublisher func([]string, string) error
	MessageSet      struct {
		Tags     []string
		Messages []Message
	}

	Message struct {
		ID   string
		Tags []string
		Data string
	}

	Plexer struct {
		batch  map[string]BatchPublisher
		single map[string]SinglePublisher
	}
)

func NewPlexer() *Plexer {
	plex := &Plexer{
		batch:  make(map[string]BatchPublisher, 0),
		single: make(map[string]SinglePublisher, 0),
	}

	return plex
}

func (plex *Plexer) AddBatcher(name string, observer BatchPublisher) {
	lumber.Trace("[PULSE :: PLEXER] Add batcher: %v...", name)
	plex.batch[name] = observer
}

func (plex *Plexer) RemoveBatcher(name string) {
	lumber.Trace("[PULSE :: PLEXER] Remove batcher: %v...", name)
	delete(plex.batch, name)
}

func (plex *Plexer) AddObserver(name string, observer SinglePublisher) {
	lumber.Trace("[PULSE :: PLEXER] Add observer: %v...", name)
	plex.single[name] = observer
}

func (plex *Plexer) RemoveObserver(name string) {
	lumber.Trace("[PULSE :: PLEXER] Remove observer: %v...", name)
	delete(plex.single, name)
}

func (plex *Plexer) Publish(messages MessageSet) error {

	for _, observer := range plex.batch {
		go observer(messages)
	}

	for _, observer := range plex.single {
		for _, message := range messages.Messages {
			message.Tags = append(message.Tags, messages.Tags...)
			go observer(append(message.Tags, message.ID), message.Data)
		}
	}
	return nil
}

func (plex *Plexer) PublishSingle(id string, tags []string, data string) error {

	messages := MessageSet{
		Tags: []string{},
		Messages: []Message{
			Message{
				ID:   id,
				Tags: tags,
				Data: data,
			},
		},
	}
	return plex.Publish(messages)
}
