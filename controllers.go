package core

import (
	"errors"
	"sync"

	"github.com/laplasd/inforo/api"

	"github.com/sirupsen/logrus"
)

type ControllerRegistry struct {
	controllers map[string]api.Controller
	mu          *sync.RWMutex
	logger      *logrus.Logger
}

type ControllerRegistryOptions struct {
	Logger *logrus.Logger
	// Другие зависимости
}

func NewControllerRegistry(opts ControllerRegistryOptions) (api.ControllerRegistry, error) {
	return &ControllerRegistry{
		mu:          &sync.RWMutex{},
		logger:      opts.Logger,
		controllers: make(map[string]api.Controller),
	}, nil

}

func (cr *ControllerRegistry) Delete(c string) error {
	return nil
}

func (cr *ControllerRegistry) Get(componentType string) (api.Controller, error) {

	if ctl, ok := cr.controllers[componentType]; ok {
		return ctl, nil
	} else {
		return nil, errors.New("controller not found")
	}
}

func (cr *ControllerRegistry) List() ([]api.Controller, error) {
	return []api.Controller{}, nil
}

func (cr *ControllerRegistry) ListType() ([]string, error) {
	// Предположим, что cr.controllers - это map[string]api.Controller
	keys := make([]string, 0, len(cr.controllers))
	for k := range cr.controllers {
		keys = append(keys, k)
	}
	return keys, nil
}

func (cr *ControllerRegistry) Register(controllerType string, controller api.Controller) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if _, exists := cr.controllers[controllerType]; exists {
		cr.logger.Warnf("Component %s already registered", controllerType)
		return errors.New("component already registered")
	}

	cr.controllers[controllerType] = controller
	//cr.logger.Infof("Registered component %s (%s) of type %s", comp.ID, comp.Name, comp.Type)
	return nil
}

func (cr *ControllerRegistry) Update(c string, comp api.Controller) error {
	return nil
}
