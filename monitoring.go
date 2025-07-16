package inforo

import (
	"errors"

	"sync"

	"github.com/laplasd/inforo/api"
	"github.com/laplasd/inforo/model"

	"github.com/sirupsen/logrus"
)

type MonitoringRegistry struct {
	monitorControllers api.MonitoringControllerRegistry
	monitorings        map[string]model.Monitoring
	*StatusManager
	*Events
	mu     *sync.RWMutex
	logger *logrus.Logger
}

type MonitoringRegistryOptions struct {
	Logger        *logrus.Logger
	Controllers   api.MonitoringControllerRegistry
	StatusManager *StatusManager
	EventManager  *Events
	// Другие зависимости
}

func NewMonitoringRegistry(opts MonitoringRegistryOptions) (api.MonitoringRegistry, error) {
	return &MonitoringRegistry{
		mu:                 &sync.RWMutex{},
		logger:             opts.Logger,
		monitorControllers: opts.Controllers,
		StatusManager:      opts.StatusManager,
		monitorings:        make(map[string]model.Monitoring),
	}, nil
}

func (mr *MonitoringRegistry) Register(tp string, m model.Monitoring) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	if _, exists := mr.monitorings[m.ID]; exists {
		return errors.New("monitoring system already registered")
	}

	controller, err := mr.monitorControllers.Get(m.Type)
	if err != nil {
		return err
	}

	err = controller.ValidateMonitoring(m.Config)
	if err != nil {
		return err
	}

	m.StatusHistory = mr.NewStatus(model.StatusPending)
	mr.monitorings[m.ID] = m
	return nil
}

func (mr *MonitoringRegistry) Get(id string) (model.Monitoring, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	m, ok := mr.monitorings[id]
	if !ok {
		return model.Monitoring{}, errors.New("monitoring system not found")
	}

	return m, nil
}

func (mr *MonitoringRegistry) Update(id string, updated model.Monitoring) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	if _, exists := mr.monitorings[id]; !exists {
		return errors.New("monitoring system not found")
	}

	updated.ID = id // не позволяем изменить ID
	mr.monitorings[id] = updated
	return nil
}

func (mr *MonitoringRegistry) Delete(id string) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	if _, exists := mr.monitorings[id]; !exists {
		return errors.New("monitoring system not found")
	}

	delete(mr.monitorings, id)
	return nil
}

func (mr *MonitoringRegistry) List() ([]model.Monitoring, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	result := make([]model.Monitoring, 0, len(mr.monitorings))
	for _, m := range mr.monitorings {
		result = append(result, m)
	}

	return result, nil
}
