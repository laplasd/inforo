package actorsystem

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/laplasd/inforo/model"
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
	tags       map[string][]string // actor path -> tags
	actorType  map[string]string   // actor path -> type
	dispatcher *Dispatcher
	logger     *logrus.Logger
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewActorSystem создает новую акторную систему
func NewActorSystem(logger *logrus.Logger) *ActorSystem {
	ctx, cancel := context.WithCancel(context.Background())
	logger.Debug("inforo: init ActorSystem")
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

// ListActors возвращает список всех зарегистрированных акторов
func (as *ActorSystem) ListActors() []string {
	as.mu.RLock()
	defer as.mu.RUnlock()

	paths := make([]string, 0, len(as.actors))
	for path := range as.actors {
		paths = append(paths, path)
	}
	return paths
}

// RegisterWithMetadata регистрирует актор с метаданными
func (as *ActorSystem) RegisterWithMetadata(path, actorType string, tags []string, actor ActorRef) error {
	if err := as.Register(path, actor); err != nil {
		return err
	}

	as.mu.Lock()
	defer as.mu.Unlock()
	as.tags[path] = tags
	as.actorType[path] = actorType
	return nil
}

// FindActorsByType ищет акторы по типу
func (as *ActorSystem) FindActorsByType(actorType string) []string {
	as.mu.RLock()
	defer as.mu.RUnlock()

	var result []string
	for path, typ := range as.actorType {
		if typ == actorType {
			result = append(result, path)
		}
	}
	return result
}

// FindActorsByTag ищет акторы по тегу
func (as *ActorSystem) FindActorsByTag(tag string) []string {
	as.mu.RLock()
	defer as.mu.RUnlock()

	var result []string
	for path, tags := range as.tags {
		for _, t := range tags {
			if t == tag {
				result = append(result, path)
				break
			}
		}
	}
	return result
}

// SetMailboxSize изменяет размер почтового ящика актора
func (as *ActorSystem) SetMailboxSize(path string, size int) error {
	actor := as.GetActor(path)
	if actor == nil {
		return ErrActorNotFound
	}

	if ba, ok := actor.(*BaseActor); ok {
		newMailbox := make(chan interface{}, size)
		close(ba.mailbox)

		ba.mu.Lock()
		ba.mailbox = newMailbox
		ba.mu.Unlock()
		return nil
	}

	return errors.New("only BaseActor supports mailbox resizing")
}

// SetDispatcherWorkers изменяет количество рабочих dispatcher
func (as *ActorSystem) SetDispatcherWorkers(count int) {
	as.dispatcher.SetWorkers(count)
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

// Сообщение для регистрации компонента
type ComponentRegisterMsg struct {
	Component model.Component
	ReplyTo   chan ComponentRegisterResponse
}

// Ответ от актора
type ComponentRegisterResponse struct {
	Component *model.Component
	Error     error
}

type ComponentReadMsg struct {
	ComponentID string
	ReplyTo     chan ComponentRegisterResponse
}

// Ответ от актора
type ComponentReadResponse struct {
	Component *model.Component
	Error     error
}
