package inforo_test

import (
	"io"
	"testing"

	"github.com/laplasd/inforo"
	"github.com/laplasd/inforo/api"
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
		logger := inforo.NewNullLogger()
		assert.NotNil(t, logger)
		assert.Equal(t, io.Discard, logger.Out)
	})
}

func TestNewDefaultCore(t *testing.T) {
	t.Run("should create core with all default registries", func(t *testing.T) {
		core := inforo.NewDefaultCore()

		assert.NotNil(t, core)
		assert.NotNil(t, core.Logger)
		assert.NotNil(t, core.Components)
		assert.NotNil(t, core.Controllers)
		assert.NotNil(t, core.Monitorings)
		assert.NotNil(t, core.MonitorControllers)
		assert.NotNil(t, core.Tasks)
		assert.NotNil(t, core.Plans)

		// Verify logger is null logger
		assert.Equal(t, io.Discard, core.Logger.Out)
	})
}

func TestNewCore(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() inforo.CoreOptions
		validate func(*testing.T, *inforo.Core)
	}{
		{
			name: "with custom logger",
			setup: func() inforo.CoreOptions {
				logger := logrus.New()
				logger.SetLevel(logrus.DebugLevel)
				return inforo.CoreOptions{Logger: logger}
			},
			validate: func(t *testing.T, c *inforo.Core) {
				assert.Equal(t, logrus.DebugLevel, c.Logger.GetLevel())
				// Other registries should still be created
				assert.NotNil(t, c.Controllers)
			},
		},
		{
			name: "with custom component registry",
			setup: func() inforo.CoreOptions {
				mockComp := new(MockComponentRegistry)
				return inforo.CoreOptions{Components: mockComp}
			},
			validate: func(t *testing.T, c *inforo.Core) {
				_, ok := c.Components.(*MockComponentRegistry)
				assert.True(t, ok)
				// Dependent registries should use this component registry
				assert.NotNil(t, c.Tasks)
			},
		},
		{
			name: "with all custom registries",
			setup: func() inforo.CoreOptions {
				return inforo.CoreOptions{
					Logger:             logrus.New(),
					Components:         new(MockComponentRegistry),
					Controllers:        new(MockControllerRegistry),
					Monitorings:        new(MockMonitoringRegistry),
					MonitorControllers: new(MockMonitoringControllerRegistry),
					Tasks:              new(MockTaskRegistry),
					Plans:              new(MockPlanRegistry),
				}
			},
			validate: func(t *testing.T, c *inforo.Core) {
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
			core := inforo.NewCore(opts)
			tt.validate(t, core)
		})
	}
}

func TestDefaultOpts(t *testing.T) {
	tests := []struct {
		name     string
		input    inforo.CoreOptions
		validate func(*testing.T, inforo.CoreOptions)
	}{
		{
			name:  "nil logger",
			input: inforo.CoreOptions{Logger: nil},
			validate: func(t *testing.T, opts inforo.CoreOptions) {
				assert.NotNil(t, opts.Logger)
				assert.Equal(t, io.Discard, opts.Logger.Out)
			},
		},
		{
			name:  "nil controllers",
			input: inforo.CoreOptions{Controllers: nil},
			validate: func(t *testing.T, opts inforo.CoreOptions) {
				assert.NotNil(t, opts.Controllers)
			},
		},
		{
			name:  "nil monitor controllers",
			input: inforo.CoreOptions{MonitorControllers: nil},
			validate: func(t *testing.T, opts inforo.CoreOptions) {
				assert.NotNil(t, opts.MonitorControllers)
			},
		},
		{
			name:  "nil components",
			input: inforo.CoreOptions{Components: nil},
			validate: func(t *testing.T, opts inforo.CoreOptions) {
				assert.NotNil(t, opts.Components)
				// Should have set the controllers dependency
				assert.NotNil(t, opts.Components.(*inforo.ComponentRegistry).Controllers)
			},
		},
		{
			name:  "nil monitorings",
			input: inforo.CoreOptions{Monitorings: nil},
			validate: func(t *testing.T, opts inforo.CoreOptions) {
				assert.NotNil(t, opts.Monitorings)
			},
		},
		{
			name:  "nil tasks",
			input: inforo.CoreOptions{Tasks: nil},
			validate: func(t *testing.T, opts inforo.CoreOptions) {
				assert.NotNil(t, opts.Tasks)
				// Should have dependencies set
				taskReg := opts.Tasks.(*inforo.TaskRegistry)
				assert.NotNil(t, taskReg.Components)
				assert.NotNil(t, taskReg.Controllers)
			},
		},
		{
			name:  "nil plans",
			input: inforo.CoreOptions{Plans: nil},
			validate: func(t *testing.T, opts inforo.CoreOptions) {
				assert.NotNil(t, opts.Plans)
				// Should have dependencies set
				planReg := opts.Plans.(*inforo.PlanRegistry)
				assert.NotNil(t, planReg.Components)
				assert.NotNil(t, planReg.Tasks)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inforo.DefaultOpts(tt.input)
			tt.validate(t, result)
		})
	}
}
