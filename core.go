package core

import (
	"io"

	"github.com/laplasd/inforo/api"
	"github.com/laplasd/inforo/model"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Core struct {
	logger             *logrus.Logger
	Components         api.ComponentRegistry
	Controllers        api.ControllerRegistry
	Monitorings        api.MonitoringRegistry
	MonitorControllers api.MonitoringControllerRegistry
	Tasks              api.TaskRegistry
	Plans              api.PlanRegistry
}

type CoreOptions struct {
	Logger             *logrus.Logger                   `json:"Logger"`
	Components         api.ComponentRegistry            `json:"Components"`
	Controllers        api.ControllerRegistry           `json:"Controllers"`
	Monitorings        api.MonitoringRegistry           `json:"Monitorings"`
	MonitorControllers api.MonitoringControllerRegistry `json:"MonitorControllers"`
	Tasks              api.TaskRegistry                 `json:"Tasks"`
	Plans              api.PlanRegistry                 `json:"Plans"`
}

func NewNullLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Out = io.Discard // Все логи будут отправляться в "никуда"
	return logger
}

func NewDefaultCore() *Core {
	opts := defaultOpts(CoreOptions{})

	c := &Core{
		logger:             opts.Logger,
		Components:         opts.Components,
		Controllers:        opts.Controllers,
		Monitorings:        opts.Monitorings,
		MonitorControllers: opts.MonitorControllers,
		Tasks:              opts.Tasks,
		Plans:              opts.Plans,
	}
	return c
}

func NewCore(opt CoreOptions) *Core {
	opts := defaultOpts(opt)

	c := &Core{
		logger:             opts.Logger,
		Components:         opts.Components,
		Controllers:        opts.Controllers,
		Monitorings:        opts.Monitorings,
		MonitorControllers: opts.MonitorControllers,
		Tasks:              opts.Tasks,
		Plans:              opts.Plans,
	}
	return c
}

func defaultOpts(opt CoreOptions) CoreOptions {
	if opt.Logger == nil {
		opt.Logger = NewNullLogger()
	}
	if opt.Controllers == nil {
		controllerOpts := ControllerRegistryOptions{
			Logger: opt.Logger,
		}
		opt.Controllers, _ = NewControllerRegistry(controllerOpts)
	}
	if opt.MonitorControllers == nil {
		monitorControllerOpts := MonitoringControllerRegistryOptions{
			Logger: opt.Logger,
		}
		opt.MonitorControllers, _ = NewMonitoringControllerRegistry(monitorControllerOpts)
	}
	if opt.Components == nil {
		componentOpts := ComponentRegistryOptions{
			Logger:      opt.Logger,
			Controllers: opt.Controllers,
		}
		opt.Components, _ = NewComponentRegistry(componentOpts)
	}
	if opt.Monitorings == nil {
		monitoringOpts := MonitoringRegistryOptions{
			Logger:      opt.Logger,
			Controllers: opt.MonitorControllers,
		}
		opt.Monitorings, _ = NewMonitoringRegistry(monitoringOpts)
	}
	if opt.Tasks == nil {
		taskOpts := TaskRegistryOptions{
			Logger:             opt.Logger,
			Components:         opt.Components,
			Controllers:        opt.Controllers,
			Monitoring:         opt.Monitorings,
			MonitorControllers: opt.MonitorControllers,
		}
		opt.Tasks, _ = NewTaskRegistry(taskOpts)
	}
	if opt.Plans == nil {
		planOpts := PlanRegistryOptions{
			Logger:     opt.Logger,
			Components: opt.Components,
			Tasks:      opt.Tasks,
		}
		opt.Plans, _ = NewPlanRegistry(planOpts)
	}
	return opt
}

func generateID(prefix string) string {
	return prefix + "-" + uuid.New().String()
}

func isValidTaskType(t model.TaskType) bool {
	switch t {
	case model.UpdateTask, model.RollbackTask, model.CheckTask:
		return true
	default:
		return false
	}
}
