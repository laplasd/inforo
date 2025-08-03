package actorsystem

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ActorRef - интерфейс для взаимодействия с актором
type ActorRef interface {
	Tell(msg interface{}) error
	Ask(msg interface{}, timeout time.Duration) (interface{}, error)
	Stop() error
	Path() string
	IsAlive() bool
}

// BaseActor - базовая реализация актора
type BaseActor struct {
	logger   logrus.Logger
	mu       sync.RWMutex
	path     string
	mailbox  chan interface{}
	behavior ActorBehavior
	stopping bool
	system   *ActorSystem
}

type ActorBehavior func(ctx ActorContext, msg interface{})

// NewBaseActor создает новый базовый актор
func NewBaseActor(path string, behavior ActorBehavior, system *ActorSystem, logger logrus.Logger) *BaseActor {
	return &BaseActor{
		logger:   logger,
		path:     path,
		mailbox:  make(chan interface{}, 1000), // Буферизированный почтовый ящик
		behavior: behavior,
		system:   system,
	}
}

func (a *BaseActor) Run() {
	a.logger.Debugf("BaseActor[%s].Run", a.path)
	for msg := range a.mailbox {
		if a.stopping {
			return
		}
		ctx := NewActorContext(context.Background(), a.path, a.system)
		a.behavior(ctx, msg)
	}
}

func (a *BaseActor) Tell(msg interface{}) error {
	a.logger.Debugf("BaseActor[%s].Tell(msg '%v')", a.path, msg)
	if a.stopping {
		return errors.New("actor is stopping")
	}
	a.mailbox <- msg
	return nil
}

func (a *BaseActor) Ask(msg interface{}, timeout time.Duration) (interface{}, error) {
	a.logger.Debugf("BaseActor[%s].Ask", a.path)
	if a.stopping {
		return nil, errors.New("actor is stopping")
	}

	respCh := make(chan interface{}, 1)
	a.mailbox <- &AskMessage{
		payload:  msg,
		response: respCh,
	}

	select {
	case resp := <-respCh:
		return resp, nil
	case <-time.After(timeout):
		return nil, errors.New("timeout waiting for response")
	}
}

func (a *BaseActor) Stop() error {
	a.logger.Debugf("BaseActor[%s].Stop", a.path)
	a.stopping = true
	close(a.mailbox)
	return nil
}

func (a *BaseActor) Path() string {
	return a.path
}

func (a *BaseActor) IsAlive() bool {
	return true
}
