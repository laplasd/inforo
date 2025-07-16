package core

import (
	"errors"
	"reflect"
	"strconv"
	"sync"

	"github.com/laplasd/inforo/model"

	"github.com/laplasd/inforo/api"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type ComponentRegistry struct {
	Controllers api.ControllerRegistry
	components  map[string]*model.Component
	Events
	StatusManager
	mu     *sync.RWMutex
	logger *logrus.Logger
}

type ComponentRegistryOptions struct {
	Logger        *logrus.Logger
	Controllers   api.ControllerRegistry
	StatusManager StatusManager
	EventManager  Events
	// Другие зависимости
}

func NewComponentRegistry(opts ComponentRegistryOptions) (api.ComponentRegistry, error) {
	return &ComponentRegistry{
		mu:            &sync.RWMutex{},
		logger:        opts.Logger,
		Controllers:   opts.Controllers,
		StatusManager: opts.StatusManager,
		components:    make(map[string]*model.Component),
	}, nil
}

func (cr *ComponentRegistry) Delete(id string) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	_, exists := cr.components[id]
	if !exists {
		return errors.New("component not found")
	}

	delete(cr.components, id)
	return nil
}

func (cr *ComponentRegistry) Get(id string) (*model.Component, error) {
	cr.logger.Debugf("ComponentRegistry.Get: call(), args: id[%s]", id)
	cr.mu.Lock()
	defer cr.mu.Unlock()

	comp, exists := cr.components[id]
	if !exists {
		return nil, errors.New("component not found")
	}
	cr.logger.Debugf("ComponentRegistry.Get: return(model.Component, error) -> (%v,%v)", comp, nil)
	return comp, nil
}

func (cr *ComponentRegistry) List() ([]*model.Component, error) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	comps := make([]*model.Component, 0, len(cr.components))
	for _, comp := range cr.components {
		comps = append(comps, comp)
	}
	return comps, nil
}

func (cr *ComponentRegistry) GetBy(key string, value string) ([]*model.Component, error) {
	cr.logger.Debugf("ComponentRegistry.GetBy: call(), args: key[%s], value[%s]", key, value)
	cr.mu.Lock()
	defer cr.mu.Unlock()

	var result []*model.Component

	for _, comp := range cr.components {
		// Используем рефлексию для доступа к полям структуры
		v := reflect.ValueOf(comp)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() != reflect.Struct {
			cr.logger.Errorf("ComponentRegistry.GetBy: component is not a struct")
			continue
		}

		field := v.FieldByName(key)
		if !field.IsValid() {
			cr.logger.Debugf("ComponentRegistry.GetBy: field %s not found in component", key)
			continue
		}

		// Преобразуем значение поля в строку для сравнения
		var fieldValue string
		switch field.Kind() {
		case reflect.String:
			fieldValue = field.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldValue = strconv.FormatInt(field.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fieldValue = strconv.FormatUint(field.Uint(), 10)
		case reflect.Bool:
			fieldValue = strconv.FormatBool(field.Bool())
		case reflect.Float32, reflect.Float64:
			fieldValue = strconv.FormatFloat(field.Float(), 'f', -1, 64)
		default:
			cr.logger.Debugf("ComponentRegistry.GetBy: unsupported field type %s for field %s", field.Kind(), key)
			continue
		}

		if fieldValue == value {
			result = append(result, comp)
		}
	}

	if len(result) == 0 {
		cr.logger.Debugf("ComponentRegistry.GetBy: no components found with %s=%s", key, value)
		return nil, errors.New("no components found")
	}

	cr.logger.Debugf("ComponentRegistry.GetBy: return([]model.Component, error) -> (%v,%v)", result, nil)
	return result, nil
}

func (cr *ComponentRegistry) Register(comp model.Component) (*model.Component, error) {
	cr.logger.Debugf("ComponentRegistry.Register: call(), args: comp[%v]", comp)

	cr.mu.Lock()
	defer cr.mu.Unlock()

	if _, exists := cr.components[comp.ID]; exists {
		cr.logger.Debugf("ComponentRegistry.Register: return(error) -> '%v'", errors.New("component already registered"))
		return nil, errors.New("component already registered")
	}
	if comp.ID == "" {
		comp.ID = uuid.New().String()
	}

	if comp.Version == "" {
		return nil, errors.New("component version is empty")
	}

	if cr.Controllers != nil {
		cr.logger.Infof("ComponentRegistry.Register: check 'metaData'")
		err := cr.checkMeta(comp.Type, comp.Metadata)
		if err != nil {
			cr.logger.Debugf("ComponentRegistry.Register: return(error) -> '%v'", err)
			return nil, err
		}
	}

	comp.StatusHistory = cr.NewStatus(model.StatusPending)
	comp.EventHistory = &model.EventHistory{}
	cr.AddEvent(comp.EventHistory, "Created component!")

	cr.components[comp.ID] = &comp
	cr.logger.Debugf("ComponentRegistry.Register: return(error) -> '%v'", nil)
	return &comp, nil
}

func (cr *ComponentRegistry) Update(id string, updatedComp *model.Component) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	comp, exists := cr.components[id]
	if !exists {
		return errors.New("component not found")
	}
	updatedComp.ID = comp.ID // чтобы не изменить ID по ошибке

	if cr.Controllers != nil {
		err := cr.checkMeta(updatedComp.Type, updatedComp.Metadata)
		if err != nil {
			return err
		}
	}

	cr.components[id] = updatedComp

	cr.logger.Infof("Component %s updated", id)
	return nil
}

func (cr *ComponentRegistry) Disable(id string) error {

	comp, err := cr.Get(id)
	if err != nil {
		return err
	}
	comp.StatusHistory = cr.NextStatus(model.StatusDisable, *comp.StatusHistory)

	err = cr.Update(comp.ID, comp)
	if err != nil {
		return err
	}
	return nil
}

func (cr *ComponentRegistry) Enable(id string) error {

	comp, err := cr.Get(id)
	if err != nil {
		return err
	}
	comp.StatusHistory = cr.NextStatus(model.StatusPending, *comp.StatusHistory)

	err = cr.Update(comp.ID, comp)
	if err != nil {
		return err
	}
	return nil

}

func (cr *ComponentRegistry) UpVersion(componentID string, version string) {

}

func (cr *ComponentRegistry) checkMeta(compType string, compMeta map[string]string) error {

	controller, err := cr.Controllers.Get(compType)
	if err != nil {
		return err
	}

	err = controller.ValideComponent(compMeta)
	if err != nil {
		return err
	}
	return nil
}
