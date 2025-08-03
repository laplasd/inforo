package actorsystem

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// SupervisorStrategy определяет поведение при падении актора
type SupervisorStrategy interface {
	HandleFailure(supervisor *Supervisor, actor ActorRef, err error)
}

// Стандартные стратегии
var (
	// RestartStrategy - перезапускает актор при падении
	RestartStrategy = &restartStrategy{}
	// StopStrategy - останавливает актор при падении
	StopStrategy = &stopStrategy{}
	// EscalateStrategy - эскалирует ошибку на уровень выше
	EscalateStrategy = &escalateStrategy{}
)

// Supervisor управляет жизненным циклом акторов
type Supervisor struct {
	mu         sync.RWMutex
	children   map[string]ActorRef
	strategies map[string]SupervisorStrategy
	logger     *logrus.Logger
	parent     *Supervisor // Родительский супервизор
}

// NewSupervisor создает новый Supervisor
func NewSupervisor(logger *logrus.Logger) *Supervisor {
	return &Supervisor{
		children:   make(map[string]ActorRef),
		strategies: make(map[string]SupervisorStrategy),
		logger:     logger,
	}
}

// AddChild регистрирует дочерний актор
func (s *Supervisor) AddChild(name string, actor ActorRef, strategy SupervisorStrategy) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.children[name] = actor
	s.strategies[name] = strategy

	// Запускаем мониторинг состояния актора
	go s.monitorActor(name, actor)
}

// monitorActor отслеживает состояние актора
func (s *Supervisor) monitorActor(name string, actor ActorRef) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !actor.IsAlive() {
				s.handleFailure(name, actor, ErrActorNotAlive)
				return
			}
		}
	}
}

// handleFailure обрабатывает сбой актора
func (s *Supervisor) handleFailure(name string, actor ActorRef, err error) {
	s.mu.RLock()
	strategy, exists := s.strategies[name]
	s.mu.RUnlock()

	if !exists {
		s.logger.Errorf("no strategy for actor %s", name)
		return
	}

	strategy.HandleFailure(s, actor, err)
}

// Реализации стратегий

type restartStrategy struct{}

func (s *restartStrategy) HandleFailure(supervisor *Supervisor, actor ActorRef, err error) {
	supervisor.logger.Warnf("restarting actor %v after error: %v", actor, err)

	// В реальной реализации здесь должна быть логика пересоздания актора
	if restartable, ok := actor.(RestartableActor); ok {
		restartable.Restart()
	}
}

type stopStrategy struct{}

func (s *stopStrategy) HandleFailure(supervisor *Supervisor, actor ActorRef, err error) {
	supervisor.logger.Warnf("stopping actor %v after error: %v", actor, err)
	actor.Stop()
}

type escalateStrategy struct{}

func (s *escalateStrategy) HandleFailure(supervisor *Supervisor, actor ActorRef, err error) {
	supervisor.logger.Errorf("escalating failure for actor %v: %v", actor, err)
	if supervisor.parent != nil {
		supervisor.parent.handleFailure("child", actor, err)
	}
}

// RestartableActor интерфейс для акторов, поддерживающих перезапуск
type RestartableActor interface {
	ActorRef
	Restart()
}
