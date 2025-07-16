package core

import (
	"io"
	"testing"

	"github.com/laplasd/inforo/api"
	"github.com/laplasd/inforo/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock registries for testing
type MockComponentRegistry struct {
	mock.Mock
	api.ComponentRegistry
}

type MockControllerRegistry struct {
	mock.Mock
	api.ControllerRegistry
}

type MockMonitoringRegistry struct {
	mock.Mock
	api.MonitoringRegistry
}

type MockMonitoringControllerRegistry struct {
	mock.Mock
	api.MonitoringControllerRegistry
}

type MockTaskRegistry struct {
	mock.Mock
	api.TaskRegistry
}

type MockPlanRegistry struct {
	mock.Mock
	api.PlanRegistry
}

func TestNewNullLogger(t *testing.T) {
	t.Run("should create logger with discard output", func(t *testing.T) {
		logger := NewNullLogger()
		assert.NotNil(t, logger)
		assert.Equal(t, io.Discard, logger.Out)
	})
}

func TestNewDefaultCore(t *testing.T) {
	t.Run("should create core with all default registries", func(t *testing.T) {
		core := NewDefaultCore()

		assert.NotNil(t, core)
		assert.NotNil(t, core.logger)
		assert.NotNil(t, core.Components)
		assert.NotNil(t, core.Controllers)
		assert.NotNil(t, core.Monitorings)
		assert.NotNil(t, core.MonitorControllers)
		assert.NotNil(t, core.Tasks)
		assert.NotNil(t, core.Plans)

		// Verify logger is null logger
		assert.Equal(t, io.Discard, core.logger.Out)
	})
}

func TestNewCore(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() CoreOptions
		validate func(*testing.T, *Core)
	}{
		{
			name: "with custom logger",
			setup: func() CoreOptions {
				logger := logrus.New()
				logger.SetLevel(logrus.DebugLevel)
				return CoreOptions{Logger: logger}
			},
			validate: func(t *testing.T, c *Core) {
				assert.Equal(t, logrus.DebugLevel, c.logger.GetLevel())
				// Other registries should still be created
				assert.NotNil(t, c.Controllers)
			},
		},
		{
			name: "with custom component registry",
			setup: func() CoreOptions {
				mockComp := new(MockComponentRegistry)
				return CoreOptions{Components: mockComp}
			},
			validate: func(t *testing.T, c *Core) {
				_, ok := c.Components.(*MockComponentRegistry)
				assert.True(t, ok)
				// Dependent registries should use this component registry
				assert.NotNil(t, c.Tasks)
			},
		},
		{
			name: "with all custom registries",
			setup: func() CoreOptions {
				return CoreOptions{
					Logger:             logrus.New(),
					Components:         new(MockComponentRegistry),
					Controllers:        new(MockControllerRegistry),
					Monitorings:        new(MockMonitoringRegistry),
					MonitorControllers: new(MockMonitoringControllerRegistry),
					Tasks:              new(MockTaskRegistry),
					Plans:              new(MockPlanRegistry),
				}
			},
			validate: func(t *testing.T, c *Core) {
				assert.IsType(t, &MockComponentRegistry{}, c.Components)
				assert.IsType(t, &MockControllerRegistry{}, c.Controllers)
				assert.IsType(t, &MockMonitoringRegistry{}, c.Monitorings)
				assert.IsType(t, &MockMonitoringControllerRegistry{}, c.MonitorControllers)
				assert.IsType(t, &MockTaskRegistry{}, c.Tasks)
				assert.IsType(t, &MockPlanRegistry{}, c.Plans)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.setup()
			core := NewCore(opts)
			tt.validate(t, core)
		})
	}
}

func TestDefaultOpts(t *testing.T) {
	tests := []struct {
		name     string
		input    CoreOptions
		validate func(*testing.T, CoreOptions)
	}{
		{
			name:  "nil logger",
			input: CoreOptions{Logger: nil},
			validate: func(t *testing.T, opts CoreOptions) {
				assert.NotNil(t, opts.Logger)
				assert.Equal(t, io.Discard, opts.Logger.Out)
			},
		},
		{
			name:  "nil controllers",
			input: CoreOptions{Controllers: nil},
			validate: func(t *testing.T, opts CoreOptions) {
				assert.NotNil(t, opts.Controllers)
			},
		},
		{
			name:  "nil monitor controllers",
			input: CoreOptions{MonitorControllers: nil},
			validate: func(t *testing.T, opts CoreOptions) {
				assert.NotNil(t, opts.MonitorControllers)
			},
		},
		{
			name:  "nil components",
			input: CoreOptions{Components: nil},
			validate: func(t *testing.T, opts CoreOptions) {
				assert.NotNil(t, opts.Components)
				// Should have set the controllers dependency
				assert.NotNil(t, opts.Components.(*ComponentRegistry).Controllers)
			},
		},
		{
			name:  "nil monitorings",
			input: CoreOptions{Monitorings: nil},
			validate: func(t *testing.T, opts CoreOptions) {
				assert.NotNil(t, opts.Monitorings)
			},
		},
		{
			name:  "nil tasks",
			input: CoreOptions{Tasks: nil},
			validate: func(t *testing.T, opts CoreOptions) {
				assert.NotNil(t, opts.Tasks)
				// Should have dependencies set
				taskReg := opts.Tasks.(*TaskRegistry)
				assert.NotNil(t, taskReg.Components)
				assert.NotNil(t, taskReg.Controllers)
			},
		},
		{
			name:  "nil plans",
			input: CoreOptions{Plans: nil},
			validate: func(t *testing.T, opts CoreOptions) {
				assert.NotNil(t, opts.Plans)
				// Should have dependencies set
				planReg := opts.Plans.(*PlanRegistry)
				assert.NotNil(t, planReg.Components)
				assert.NotNil(t, planReg.Tasks)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultOpts(tt.input)
			tt.validate(t, result)
		})
	}
}

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		validate func(*testing.T, string)
	}{
		{
			name:   "with prefix",
			prefix: "test",
			validate: func(t *testing.T, id string) {
				assert.Contains(t, id, "test-")
				assert.Greater(t, len(id), len("test-")+10) // UUID part should be long enough
			},
		},
		{
			name:   "empty prefix",
			prefix: "",
			validate: func(t *testing.T, id string) {
				assert.NotContains(t, id, "--") // Shouldn't have double dash
				assert.Greater(t, len(id), 10)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := generateID(tt.prefix)
			tt.validate(t, id)
		})
	}
}

func TestIsValidTaskType(t *testing.T) {
	tests := []struct {
		name     string
		input    model.TaskType
		expected bool
	}{
		{"valid update task", model.UpdateTask, true},
		{"valid rollback task", model.RollbackTask, true},
		{"valid check task", model.CheckTask, true},
		{"invalid task type", model.TaskType("invalid"), false},
		{"empty task type", model.TaskType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidTaskType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
