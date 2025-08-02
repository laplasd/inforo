package actors

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	ErrActorNotAlive       = errors.New("actor is not alive")
	ErrActorTimeout        = errors.New("actor response timeout")
	ErrActorMailboxFull    = errors.New("actor mailbox is full")
	ErrActorAlreadyExists  = errors.New("actor with this name already exists")
	ErrActorNotFound       = errors.New("actor not found")
	ErrActorInvalidMessage = errors.New("invalid message type")
)

// ActorSystem - система управления акторами
type ActorSystem struct {
	mu         sync.RWMutex
	actors     map[string]ActorRef
	dispatcher *Dispatcher
	logger     *logrus.Logger
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewActorSystem создает новую акторную систему
func NewActorSystem(logger *logrus.Logger) *ActorSystem {
	ctx, cancel := context.WithCancel(context.Background())
	return &ActorSystem{
		actors:     make(map[string]ActorRef),
		dispatcher: NewDispatcher(100, logger),
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Register регистрирует актор в системе
func (as *ActorSystem) Register(path string, actor ActorRef) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if _, exists := as.actors[path]; exists {
		return fmt.Errorf("actor %s already exists", path)
	}

	as.actors[path] = actor
	if ba, ok := actor.(*BaseActor); ok {
		as.dispatcher.Schedule(ba.Run)
	}
	return nil
}

// GetActor возвращает ссылку на актор
func (as *ActorSystem) GetActor(path string) ActorRef {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.actors[path]
}

// Tell отправляет сообщение без ожидания ответа
func (as *ActorSystem) Tell(path string, msg interface{}) error {
	actor := as.GetActor(path)
	if actor == nil {
		return fmt.Errorf("actor %s not found", path)
	}
	return actor.Tell(msg)
}

// Ask отправляет сообщение с ожиданием ответа
func (as *ActorSystem) Ask(path string, msg interface{}, timeout time.Duration) (interface{}, error) {
	actor := as.GetActor(path)
	if actor == nil {
		return nil, fmt.Errorf("actor %s not found", path)
	}
	return actor.Ask(msg, timeout)
}

// Stop останавливает акторную систему
func (as *ActorSystem) Stop() {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.cancel()
	for _, actor := range as.actors {
		actor.Stop()
	}
	as.actors = nil
}
