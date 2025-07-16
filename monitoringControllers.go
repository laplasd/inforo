package inforo

import (
	"errors"
	"sync"

	"github.com/laplasd/inforo/api"

	"github.com/sirupsen/logrus"
)

type MonitoringControllerRegistry struct {
	сontrollers map[string]api.MonitoringController
	mu          *sync.RWMutex
	Logger      *logrus.Logger
}

type MonitoringControllerRegistryOptions struct {
	Logger *logrus.Logger
	// Другие зависимости
}

func NewMonitoringControllerRegistry(opts MonitoringControllerRegistryOptions) (api.MonitoringControllerRegistry, error) {
	return &MonitoringControllerRegistry{
		mu:          &sync.RWMutex{},
		Logger:      opts.Logger,
		сontrollers: make(map[string]api.MonitoringController),
	}, nil

}

func (mcr *MonitoringControllerRegistry) Delete(c string) error {
	return nil
}

func (mcr *MonitoringControllerRegistry) Get(componentType string) (api.MonitoringController, error) {

	if ctl, ok := mcr.сontrollers[componentType]; ok {
		return ctl, nil
	} else {
		return nil, errors.New("controller not found")
	}
}

func (mcr *MonitoringControllerRegistry) List() ([]api.MonitoringController, error) {
	return []api.MonitoringController{}, nil
}

func (mcr *MonitoringControllerRegistry) Register(controllerType string, controller api.MonitoringController) error {
	mcr.mu.Lock()
	defer mcr.mu.Unlock()

	if _, exists := mcr.сontrollers[controllerType]; exists {
		mcr.Logger.Warnf("Component %s already registered", controllerType)
		return errors.New("component already registered")
	}

	mcr.сontrollers[controllerType] = controller
	//cr.logger.Infof("Registered component %s (%s) of type %s", comp.ID, comp.Name, comp.Type)
	return nil
}

func (mcr *MonitoringControllerRegistry) Update(c string, comp api.MonitoringController) error {
	return nil
}
