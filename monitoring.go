package inforo

import (
	"errors"

	"sync"

	"github.com/google/uuid"
	"github.com/laplasd/inforo/api"
	"github.com/laplasd/inforo/model"

	"github.com/sirupsen/logrus"
)

type MonitoringRegistry struct {
	monitorControllers api.MonitoringControllerRegistry
	monitorings        map[string]*model.Monitoring
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

	mr := &MonitoringRegistry{
		mu:                 &sync.RWMutex{},
		logger:             opts.Logger,
		monitorControllers: opts.Controllers,
		StatusManager:      opts.StatusManager,
		monitorings:        make(map[string]*model.Monitoring),
	}
	// mr.Register("promql-monitor", controllers.NewPromQLMonitorController(opts.Logger, "http://prometheus:9090/api/v1"))
	return mr, nil
}

func (mr *MonitoringRegistry) Register(tp string, m *model.Monitoring) (*model.Monitoring, error) {
	mr.logger.Debugf("ComponentRegistry.Register: call(), args: m[%v]", m)

	if m.ID == "" {
		m.ID = uuid.New().String()
	}

	if mr.monitorControllers != nil {
		mr.logger.Infof("MonitoringRegistry.Register: check 'metaData'")
		err := mr.checkConfig(m.Type, m.Config)
		if err != nil {
			mr.logger.Debugf("ComponentRegistry.Register: return(error) -> '%v'", err)
			return nil, err
		}
	}

	mr.mu.Lock()
	defer mr.mu.Unlock()

	if _, exists := mr.monitorings[m.ID]; exists {
		return nil, errors.New("monitoring system already registered")
	}

	m.StatusHistory = mr.NewStatus(model.StatusPending)
	m.EventHistory = &model.EventHistory{}
	mr.AddEvent(m.EventHistory, "Created monitoring!")

	m.StatusHistory = mr.NewStatus(model.StatusPending)
	mr.monitorings[m.ID] = m
	return m, nil
}

func (mr *MonitoringRegistry) Get(id string) (*model.Monitoring, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	m, ok := mr.monitorings[id]
	if !ok {
		return nil, errors.New("monitoring system not found")
	}

	return m, nil
}

func (mr *MonitoringRegistry) Update(id string, updated *model.Monitoring) error {
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

func (mr *MonitoringRegistry) List() ([]*model.Monitoring, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	result := make([]*model.Monitoring, 0, len(mr.monitorings))
	for _, m := range mr.monitorings {
		result = append(result, m)
	}

	return result, nil
}

func (mr *MonitoringRegistry) checkConfig(compType string, compConfig map[string]string) error {

	controller, err := mr.monitorControllers.Get(compType)
	if err != nil {
		return err
	}

	err = controller.ValidateMonitoring(compConfig)
	if err != nil {
		return err
	}
	return nil
}
