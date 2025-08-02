package inforo_test

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/laplasd/inforo"
	"github.com/laplasd/inforo/api"
	"github.com/laplasd/inforo/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Warn(args ...interface{}) {
	m.Called(args...)
}

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

func NewTestDefaultCore() *inforo.Core {
	opts := inforo.DefaultOpts(inforo.CoreOptions{
		Logger: NewTestLogger(), // Используем специальный логгер для тестов
	})
	return inforo.NewCore(opts)
}

func NewTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel) // Включаем все уровни для тестов
	logger.Formatter = &logrus.TextFormatter{
		DisableTimestamp: true, // Упрощаем проверку
	}
	return logger
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

func TestCore_EventSystem(t *testing.T) {
	t.Run("should emit and receive events", func(t *testing.T) {
		core := NewTestDefaultCore()
		eventChan := core.Subscribe()

		testEvent := model.StreamEvent{
			Type:    "TestEvent",
			Payload: "test payload",
		}

		core.EmitEvent(testEvent)

		select {
		case received := <-eventChan:
			assert.Equal(t, testEvent.Type, received.Type)
			assert.Equal(t, testEvent.Payload, received.Payload)
		case <-time.After(100 * time.Millisecond):
			assert.Fail(t, "Event not received")
		}
	})

	t.Run("should handle event overflow", func(t *testing.T) {
		// 1. Создаем ядро с очень маленьким буфером
		core := NewTestDefaultCore()

		// 2. Перенаправляем вывод в буфер
		var buf bytes.Buffer
		core.Logger.SetOutput(&buf)

		// 4. Вызываем переполнение
		for i := 0; i < 101; i++ {
			core.EmitEvent(model.StreamEvent{Type: "Test"})
		}

		// 5. Даём время на обработку
		time.Sleep(10 * time.Millisecond)

		// 6. Проверяем логи
		logOutput := buf.String()
		assert.Contains(t, logOutput, "level=warning msg=\"Event stream overflow, event dropped\"",
			"Expected overflow message in logs. Got: %s", logOutput)
	})

	t.Run("should support multiple subscribers", func(t *testing.T) {
		core := NewTestDefaultCore()
		sub1 := core.Subscribe()
		sub2 := core.Subscribe()

		testEvent := model.StreamEvent{Type: "MultiSubscriber"}
		core.EmitEvent(testEvent)

		var wg sync.WaitGroup
		wg.Add(2)

		compareEvent := func(expected, actual model.StreamEvent) {
			assert.Equal(t, expected.Type, actual.Type)
			assert.Equal(t, expected.Payload, actual.Payload)
			assert.Equal(t, expected.Origin, actual.Origin)
			// Поле Timestamp игнорируем
		}

		go func() {
			defer wg.Done()
			compareEvent(testEvent, <-sub1)
		}()

		go func() {
			defer wg.Done()
			compareEvent(testEvent, <-sub2)
		}()

		wg.Wait()
	})
}

func TestCore_EventFiltering(t *testing.T) {
	t.Run("should filter events by type", func(t *testing.T) {
		core := NewTestDefaultCore()
		filteredChan := core.Subscribe("TypeA", "TypeB")

		typeAEvent := model.StreamEvent{Type: "TypeA", Payload: "A"}
		typeBEvent := model.StreamEvent{Type: "TypeB", Payload: "B"}
		typeCEvent := model.StreamEvent{Type: "TypeC", Payload: "C"}

		core.EmitEvent(typeAEvent)
		core.EmitEvent(typeBEvent)
		core.EmitEvent(typeCEvent)

		received := make([]model.StreamEvent, 0)
		timeout := time.After(500 * time.Millisecond)

		for i := 0; i < 2; i++ {
			select {
			case e := <-filteredChan:
				received = append(received, e)
			case <-timeout:
				break
			}
		}

		assert.Len(t, received, 2)

		// Сравниваем только Type и Payload, игнорируя Timestamp
		containsEvent := func(events []model.StreamEvent, want model.StreamEvent) bool {
			for _, e := range events {
				if e.Type == want.Type && e.Payload == want.Payload {
					return true
				}
			}
			return false
		}

		assert.True(t, containsEvent(received, typeAEvent), "Expected event TypeA not found")
		assert.True(t, containsEvent(received, typeBEvent), "Expected event TypeB not found")
		assert.False(t, containsEvent(received, typeCEvent), "Unexpected event TypeC found")
	})

	t.Run("should handle event handlers", func(t *testing.T) {
		core := NewTestDefaultCore()
		var handlerCalled bool

		core.On("CustomEvent", func(e model.StreamEvent) {
			handlerCalled = true
			assert.Equal(t, "CustomEvent", e.Type)
			assert.Equal(t, "data", e.Payload)
		})

		core.EmitEvent(model.StreamEvent{
			Type:    "CustomEvent",
			Payload: "data",
		})

		// Даем время на обработку
		time.Sleep(10 * time.Millisecond)
		assert.True(t, handlerCalled)
	})
}

func TestCore_Shutdown(t *testing.T) {
	t.Run("should close subscriber channels on shutdown", func(t *testing.T) {
		core := NewTestDefaultCore()
		sub := core.Subscribe()

		go func() {
			time.Sleep(5 * time.Millisecond)
			core.Shutdown()
		}()

		// Канал должен закрыться
		_, ok := <-sub
		assert.False(t, ok)
	})
}

func TestCore_ConcurrentEvents(t *testing.T) {
	t.Run("should handle concurrent event emission", func(t *testing.T) {
		core := NewTestDefaultCore()
		eventChan := core.Subscribe()

		var wg sync.WaitGroup
		count := 80

		wg.Add(count)
		for i := 0; i < count; i++ {
			go func(n int) {
				defer wg.Done()
				core.EmitEvent(model.StreamEvent{
					Type:    "Concurrent",
					Payload: n,
				})
			}(i)
		}

		wg.Wait()

		received := make(map[int]bool)
		for i := 0; i < count; i++ {
			select {
			case e := <-eventChan:
				received[e.Payload.(int)] = true
			case <-time.After(500 * time.Millisecond):
				break
			}
		}

		assert.Len(t, received, count)
		for i := 0; i < count; i++ {
			assert.True(t, received[i])
		}
	})
}

func TestCore_EventTimestamps(t *testing.T) {
	t.Run("should set proper timestamps", func(t *testing.T) {
		core := NewTestDefaultCore()
		eventChan := core.Subscribe()

		before := time.Now()
		core.EmitEvent(model.StreamEvent{Type: "TimestampTest"})
		time.Sleep(5 * time.Millisecond)
		select {
		case e := <-eventChan:
			assert.True(t, e.Timestamp.After(before) || e.Timestamp.Equal(before))
			assert.True(t, e.Timestamp.Before(time.Now()))
		case <-time.After(100 * time.Millisecond):
			assert.Fail(t, "Event not received")
		}
	})
}
