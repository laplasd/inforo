package inforo

import (
	"errors"
	"fmt"
	"sync"

	"github.com/laplasd/inforo/api"
	"github.com/laplasd/inforo/model"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type TaskRegistry struct {
	tasks              map[string]*model.Task
	Components         api.ComponentRegistry
	Controllers        api.ControllerRegistry
	Monitoring         api.MonitoringRegistry
	MonitorControllers api.MonitoringControllerRegistry
	*StatusManager
	*Events
	MU     *sync.RWMutex
	logger *logrus.Logger
}

type TaskRegistryOptions struct {
	Logger             *logrus.Logger
	Components         api.ComponentRegistry
	Controllers        api.ControllerRegistry
	Monitoring         api.MonitoringRegistry
	MonitorControllers api.MonitoringControllerRegistry
	StatusManager      *StatusManager
	EventManager       *Events
}

func NewTaskRegistry(opts TaskRegistryOptions) (api.TaskRegistry, error) {
	return &TaskRegistry{
		MU:                 &sync.RWMutex{},
		Components:         opts.Components,
		Controllers:        opts.Controllers,
		Monitoring:         opts.Monitoring,
		MonitorControllers: opts.MonitorControllers,
		logger:             opts.Logger,
		StatusManager:      opts.StatusManager,
		Events:             opts.EventManager,
		tasks:              make(map[string]*model.Task),
	}, nil
}

func (ts *TaskRegistry) Validate(task *model.Task) error {
	if len(task.Components) == 0 {
		return errors.New("Components list is empty")
	}

	if task.RollBack != nil {
		if task.RollBack.Type == "" {
			return fmt.Errorf("rollback type not found")
		}
		if (task.Components != nil && task.Metadata == nil) || (task.Components == nil && task.Metadata != nil) {
			return errors.New("both Components and Metadata must be either set or not set together")
		}
	}

	for _, depends := range task.Components {
		_, err := ts.Components.Get(depends)
		if err != nil {
			return err
		}
	}
	if task.PreChecks != nil {
		if err := ts.IsValidCheck(task.PreChecks); err != nil {
			return err
		}
	}

	if !isValidTaskType(task.Type) {
		return errors.New("invalid task type")
	}

	for _, depends := range task.DependsOn {
		if _, exists := ts.tasks[depends.ID]; !exists {
			return fmt.Errorf("Dependency '%s' not found", depends.ID)
		}
	}
	return nil
}

func (ts *TaskRegistry) Register(task *model.Task) (*model.Task, error) {
	ts.MU.Lock()
	defer ts.MU.Unlock()

	if _, exists := ts.tasks[task.ID]; exists {
		return nil, errors.New("task already exists")
	}

	if err := ts.Validate(task); err != nil {
		return nil, err
	}

	fullTask := &model.Task{
		ID:            task.ID,
		Name:          task.Name,
		Type:          task.Type,
		Components:    task.Components,
		DependsOn:     task.DependsOn,
		PreChecks:     task.PreChecks,
		PostChecks:    task.PostChecks,
		StatusHistory: ts.NewStatus(model.StatusCreated),
		EventHistory:  &model.EventHistory{},
		Metadata:      task.Metadata,
		RollBack:      task.RollBack,
	}
	ts.AddEvent(fullTask.EventHistory, "Created task!")

	ts.tasks[fullTask.ID] = fullTask
	return fullTask, nil
}

func (ts *TaskRegistry) Get(taskID string) (*model.Task, error) {
	ts.logger.Debugf("TaskRegistry.Get() - taskID: %s", taskID)
	ts.MU.Lock()
	defer ts.MU.Unlock()

	task, exists := ts.tasks[taskID]
	if !exists {
		ts.logger.Errorf("TaskRegistry.Get() - not found task with ID: %s", taskID)
		return nil, fmt.Errorf("task with ID '%s' not found", taskID)
	}
	ts.logger.Debugf("TaskRegistry.Get() - found task with ID: %s", taskID)
	return task, nil
}

func (ts *TaskRegistry) Update(id string, updated *model.Task) error {
	ts.MU.Lock()
	defer ts.MU.Unlock()

	task, exists := ts.tasks[id]
	if !exists {
		return errors.New("task not found")
	}

	// Обновляем поля по одному, чтобы не потерять статус и другие важные данные
	if updated.Name != "" {
		task.Name = updated.Name
	}
	if updated.Type != "" {
		task.Type = updated.Type
	}
	if len(updated.Components) != 0 {
		task.Components = updated.Components
	}
	if updated.Metadata != nil {
		task.Metadata = updated.Metadata
	}
	if updated.DependsOn != nil {
		task.DependsOn = updated.DependsOn
	}
	if updated.PreChecks != nil {
		task.PreChecks = updated.PreChecks
	}
	if updated.PostChecks != nil {
		task.PostChecks = updated.PostChecks
	}
	if updated.StatusHistory != nil {
		task.StatusHistory = updated.StatusHistory
	}

	// сохраняем обновлённую задачу обратно в мапу (по сути не обязательно, тк task — ссылка)
	ts.tasks[id] = task

	return nil
}

func (ts *TaskRegistry) Delete(id string) error {
	ts.MU.Lock()
	defer ts.MU.Unlock()

	if _, exists := ts.tasks[id]; !exists {
		return errors.New("task not found")
	}
	delete(ts.tasks, id)
	return nil
}

func (ts *TaskRegistry) List() ([]*model.Task, error) {
	ts.MU.Lock()
	defer ts.MU.Unlock()

	tasks := make([]*model.Task, 0, len(ts.tasks))
	for _, t := range ts.tasks {
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (ts *TaskRegistry) ForkAsync(taskID string, executionID string) (string, error) {

	if executionID == "" {
		executionID = uuid.New().String()
	}
	ts.logger.Debugf("[%s] TaskRegistry.ForkAsync() - taskID: %s", executionID, taskID)

	// Проверяем существование задачи без блокировки async части
	ts.MU.RLock()
	_, exists := ts.tasks[taskID]
	ts.MU.RUnlock()

	if !exists {
		return "", fmt.Errorf("task not found")
	}

	go ts.Fork(taskID, executionID)
	ts.logger.Debugf("[%s] TaskRegistry.ForkAsync() - started forkAsync()", executionID)

	return executionID, nil
}

func (ts *TaskRegistry) Fork(taskID string, executionID string) (string, error) {

	if executionID == "" {
		executionID = uuid.New().String()
	}

	ts.logger.Debugf("[%s] TaskRegistry.Fork() - taskID: %s", executionID, taskID)

	// Первым делом сообщаем о статусе запуска
	task, err := ts.prepareTask(taskID)
	if err != nil {
		return "", err
	}
	ts.AddEvent(task.EventHistory, "Fork task!")
	// Регистрируем выполнение в registry (нужно реализовать этот функционал)
	ts.registerExecution(executionID, taskID)
	defer ts.unregisterExecution(executionID)

	// Обрабатываем зависимости
	ts.logger.Debugf("[%s] TaskRegistry.Fork() - DependsOn: %c", executionID, len(task.DependsOn))
	if len(task.DependsOn) != 0 {
		err = ts.resolveDepens(task.DependsOn, executionID)
	}
	if err != nil {
		return "", err
	}

	err = ts.UpdateTaskStatus(task, model.StatusRunning)
	if err != nil {
		return "", err
	}
	ts.AddEvent(task.EventHistory, "Running task!")

	if task.PreChecks != nil {
		err = ts.runChecks(task.PreChecks)
		if err != nil {
			ts.UpdateTaskStatus(task, model.StatusFailed)
			return "", err
		}
	}

	type TaskComponent struct {
		Component  *model.Component
		Controller api.Controller
	}
	components := make([]TaskComponent, 0, len(task.Components))
	for _, component := range task.Components {
		component, err := ts.Components.Get(component)
		if err != nil {
			ts.UpdateTaskStatus(task, model.StatusFailed)
			return "", err
		}
		controller, err := ts.Controllers.Get(component.Type)
		if err != nil {
			ts.UpdateTaskStatus(task, model.StatusFailed)
			return "", err
		}
		components = append(components, TaskComponent{
			Component:  component,
			Controller: controller,
		})
	}

	for _, tc := range components {
		err = tc.Controller.RunTask(task.Metadata, tc.Component.Metadata)
		if err != nil {
			ts.UpdateTaskStatus(task, model.StatusFailed)
			return "", err
		}

		err = ts.UpdateTaskStatus(task, model.StatusSuccess)
		if err != nil {
			return "", err
		}
		ts.AddEvent(task.EventHistory, "Success task!")
	}

	if task.PostChecks != nil {
		err = ts.runChecks(task.PostChecks)
		if err != nil {
			ts.UpdateTaskStatus(task, model.StatusFailed)
			return "", err
		}
	}

	return "", nil
}

func (ts *TaskRegistry) resolveDepens(DependsOn []model.Depends, executionID string) error {

	for _, depends := range DependsOn {
		ts.logger.Debugf("[%s] TaskRegistry.Fork() - DependsType: %s, DependsID: %s", executionID, depends.Type, depends.ID)
		task, err := ts.Get(depends.ID)
		if err != nil {
			ts.UpdateTaskStatus(task, model.StatusFailed)
			return err
		}

		switch depends.Type {

		case model.Ordered:
			if task.StatusHistory.LastStatus == model.StatusSuccess {
				continue
			}
			ts.AddEvent(task.EventHistory, "Triggered by DependsOn!")
			_, err = ts.Fork(task.ID, executionID)
			if err != nil {
				ts.UpdateTaskStatus(task, model.StatusFailed)
				return err
			}
		case model.Blocking:
			if task.StatusHistory.LastStatus == model.StatusSuccess {
				continue
			} else {

			}

		case model.Advisory:

		case model.Strict:
			return fmt.Errorf("strict depens")

		}

	}

	return nil
}

func (ts *TaskRegistry) runChecks(checks []*model.Check) error {
	for _, check := range checks {
		monitoring, err := ts.Monitoring.Get(check.MonitoringID)
		if err != nil {
			return err
		}
		controller, err := ts.MonitorControllers.Get(monitoring.Type)
		if err != nil {
			return err
		}
		controller.RunCheck(check.Metadata)
	}
	return nil
}

// Дополнительные методы для управления выполнениями
func (ts *TaskRegistry) registerExecution(executionID string, taskID string) {
}

func (ts *TaskRegistry) unregisterExecution(executionID string) {
}

func (ts *TaskRegistry) RollBackAsync(taskID string, executionID string) (string, error) {

	if executionID == "" {
		executionID = uuid.New().String()
	}
	ts.logger.Debugf("[%s] TaskRegistry.RollBackAsync() - taskID: %s", executionID, taskID)

	// Проверяем существование задачи без блокировки async части
	ts.MU.RLock()
	task, exists := ts.tasks[taskID]
	ts.MU.RUnlock()

	if !exists {
		return "", fmt.Errorf("task not found")
	}

	go func() {
		_, err := ts.RollBack(taskID, executionID)
		if err != nil {
			ts.logger.Errorf("[%s] TaskRegistry.RollBackAsync() - RollBack failed: %v", executionID, err)
			ts.AddEvent(task.EventHistory, err.Error())
		} else {
			ts.logger.Debugf("[%s] TaskRegistry.RollBackAsync() - RollBack completed successfully", executionID)
		}
	}()
	ts.logger.Debugf("[%s] TaskRegistry.RollBackAsync() - started RollBack()", executionID)

	return executionID, nil
}

func (ts *TaskRegistry) RollBack(taskID string, executionID string) (string, error) {
	ts.logger.Debugf("[%s] TaskRegistry.RollBack() - taskID: %s", executionID, taskID)

	task, err := ts.Get(taskID)
	if err != nil {
		return "", err
	}
	ts.AddEvent(task.EventHistory, "Rolling back task...")
	ts.logger.Debugf("[%s] TaskRegistry.RollBack() - check struct", executionID)

	ts.logger.Debugf("[%s] TaskRegistry.RollBack() - check Components", executionID)

	for _, componentID := range task.Components {
		ts.logger.Debugf("[%s] TaskRegistry.RollBack() - componentID: %s", executionID, componentID)
		component, err := ts.Components.Get(componentID)
		if err != nil {
			ts.UpdateTaskStatus(task, model.StatusFailed)
			return "", err
		}

		controller, err := ts.Controllers.Get(component.Type)
		if err != nil {
			ts.UpdateTaskStatus(task, model.StatusFailed)
			return "", err
		}
		err = controller.RunTask(task.RollBack.Metadata, component.Metadata)
		if err != nil {
			ts.UpdateTaskStatus(task, model.StatusFailed)
			return "", err
		}

		err = ts.UpdateTaskStatus(task, model.StatusRollBack)
		if err != nil {
			return "", err
		}
		ts.AddEvent(task.EventHistory, "RollBack task!")

	}
	return executionID, nil
}

func (ts *TaskRegistry) Status(taskID string) (string, error) {
	return "", nil
}
func (ts *TaskRegistry) Stop(taskID string) error {
	return nil
}
func (ts *TaskRegistry) Pause(taskID string) error {

	return nil
}

func (ts *TaskRegistry) IsValidCheck(checks []*model.Check) error {
	return nil
}

func (ts *TaskRegistry) UpdateTaskStatus(task *model.Task, status model.Status) error {
	task.MU.Lock()
	defer task.MU.Unlock()

	// Создаем копию задачи для изменения
	updatedTask := task
	updatedTask.StatusHistory = ts.NextStatus(status, *task.StatusHistory)
	ts.tasks[task.ID] = updatedTask
	ts.logger.Debugf("TaskRegistry.updateTaskStatus() - task.ID: %s, last_status -> %s", task.ID, status)
	return nil
}

func (ts *TaskRegistry) prepareTask(taskID string) (*model.Task, error) {
	ts.logger.Debugf("TaskRegistry.prepareTask() - call")

	task, err := ts.Get(taskID)
	if err != nil {
		return nil, err
	}
	ts.logger.Debugf("TaskRegistry.prepareTask() - taskID: %s", taskID)

	err = ts.UpdateTaskStatus(task, model.StatusPending)
	if err != nil {
		return nil, err
	}
	ts.AddEvent(task.EventHistory, "Checking task!")
	return task, nil
}
