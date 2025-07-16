// Package inforo provides the core orchestration functionality for managing components,
// tasks, monitoring, and execution plans in a distributed system.
//
// The package offers:
// - Component registration and management
// - Task execution workflows
// - Monitoring integration
// - Plan creation and execution

package inforo

import (
	"io"

	"github.com/laplasd/inforo/api"
	"github.com/laplasd/inforo/model"

	"github.com/sirupsen/logrus"
)

// Core represents the central orchestrator that manages all system operations.
// It contains registries for different system aspects and coordinates their interactions.
type Core struct {
	Logger             *logrus.Logger                   // Central logger instance
	Components         api.ComponentRegistry            // Registry for system components
	Controllers        api.ControllerRegistry           // Registry for component controllers
	Monitorings        api.MonitoringRegistry           // Registry for monitoring systems
	MonitorControllers api.MonitoringControllerRegistry // Registry for monitoring controllers
	Tasks              api.TaskRegistry                 // Registry for task management
	Plans              api.PlanRegistry                 // Registry for execution plans
}

// CoreOptions provides configuration options for initializing a Core instance.
// All fields are optional - nil values will be replaced with default implementations.
type CoreOptions struct {
	Logger             *logrus.Logger                   `json:"Logger"`             // Custom logger instance
	Components         api.ComponentRegistry            `json:"Components"`         // Custom component registry
	Controllers        api.ControllerRegistry           `json:"Controllers"`        // Custom controller registry
	Monitorings        api.MonitoringRegistry           `json:"Monitorings"`        // Custom monitoring registry
	MonitorControllers api.MonitoringControllerRegistry `json:"MonitorControllers"` // Custom monitoring controller registry
	Tasks              api.TaskRegistry                 `json:"Tasks"`              // Custom task registry
	Plans              api.PlanRegistry                 `json:"Plans"`              // Custom plan registry
}

// NewNullLogger creates a logger that discards all log output.
// Useful for testing or when logging is not required.
//
// Returns:
// *logrus.Logger - configured logger with output discarded
func NewNullLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Out = io.Discard
	return logger
}

// NewDefaultCore creates a new Core instance with default implementations
// for all registries and a null logger.
//
// Returns:
// *Core - initialized core instance with default configurations
func NewDefaultCore() *Core {
	opts := DefaultOpts(CoreOptions{})

	c := &Core{
		Logger:             opts.Logger,
		Components:         opts.Components,
		Controllers:        opts.Controllers,
		Monitorings:        opts.Monitorings,
		MonitorControllers: opts.MonitorControllers,
		Tasks:              opts.Tasks,
		Plans:              opts.Plans,
	}
	return c
}

// NewCore creates a new Core instance with custom configurations.
// Any nil options will be replaced with default implementations.
//
// Parameters:
//   - opt: CoreOptions containing custom configurations
//
// Returns:
// *Core - initialized core instance with merged custom and default configurations
func NewCore(opt CoreOptions) *Core {
	opts := DefaultOpts(opt)

	c := &Core{
		Logger:             opts.Logger,
		Components:         opts.Components,
		Controllers:        opts.Controllers,
		Monitorings:        opts.Monitorings,
		MonitorControllers: opts.MonitorControllers,
		Tasks:              opts.Tasks,
		Plans:              opts.Plans,
	}
	return c
}

// DefaultOpts ensures all required options are set with default values
// where no custom implementation is provided.
//
// Parameters:
//   - opt: CoreOptions containing partial or complete configurations
//
// Returns:
// CoreOptions - complete options with defaults filled in
func DefaultOpts(opt CoreOptions) CoreOptions {
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

// isValidTaskType checks if a task type is valid and supported by the system.
//
// Parameters:
//   - t: model.TaskType to validate
//
// Returns:
// bool - true if the task type is valid, false otherwise
func isValidTaskType(t model.TaskType) bool {
	switch t {
	case model.UpdateTask, model.RollbackTask, model.CheckTask:
		return true
	default:
		return false
	}
}
