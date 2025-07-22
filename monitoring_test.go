package inforo_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/laplasd/inforo"
	"github.com/laplasd/inforo/api"
	"github.com/laplasd/inforo/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

/*/ MockMonitoringControllerRegistry для тестирования
type MockMonitoringControllerRegistry struct {
	mock.Mock
}
*/

func (m *MockMonitoringControllerRegistry) Get(tp string) (api.MonitoringController, error) {
	args := m.Called(tp)

	// Явно проверяем, возвращаем ли мы nil интерфейс
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(api.MonitoringController), args.Error(1)
}

func (m *MockMonitoringControllerRegistry) Register(tp string, c api.MonitoringController) error {
	m.Called(tp, c)
	return nil
}

// MockMonitoringController для тестирования
type MockMonitoringController struct {
	mock.Mock
}

func (m *MockMonitoringController) ValidateMonitoring(config map[string]string) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockMonitoringController) CheckMonitoring(config map[string]string) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockMonitoringController) RunCheck(config map[string]string) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockMonitoringController) ValidateCheck(config map[string]string) error {
	args := m.Called(config)
	return args.Error(0)
}

func NewTestDefaultMonitoringRegistry() api.MonitoringRegistry {
	core := NewTestDefaultCore()
	return core.Monitorings
}

func setupTestRegistry(t *testing.T) (*inforo.MonitoringRegistry, *MockMonitoringControllerRegistry) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Уменьшаем логирование для тестов

	mockControllers := new(MockMonitoringControllerRegistry)

	opts := inforo.MonitoringRegistryOptions{
		Logger:        logger,
		Controllers:   mockControllers,
		StatusManager: &inforo.StatusManager{},
		EventManager:  &inforo.Events{},
	}
	registry, err := inforo.NewMonitoringRegistry(opts)
	assert.NoError(t, err)

	return registry.(*inforo.MonitoringRegistry), mockControllers
}

func TestRegister(t *testing.T) {
	t.Run("successful registration with generated ID", func(t *testing.T) {
		registry, mockControllers := setupTestRegistry(t)
		monitoring := &model.Monitoring{Type: "prometheus"}

		// Настраиваем мок
		mockController := new(MockMonitoringController)
		mockController.On("ValidateMonitoring", monitoring.Config).Return(nil)
		mockControllers.On("Get", monitoring.Type).Return(mockController, nil)

		result, err := registry.Register(monitoring.Type, monitoring)
		assert.NoError(t, err)
		assert.NotEmpty(t, result.ID)
		assert.Equal(t, model.StatusPending, result.StatusHistory.LastStatus)

		mockController.AssertExpectations(t)
		mockControllers.AssertExpectations(t)
	})

	t.Run("successful registration with existing ID", func(t *testing.T) {
		registry, mockControllers := setupTestRegistry(t)
		id := uuid.New().String()
		monitoring := &model.Monitoring{ID: id, Type: "prometheus"}

		mockController := new(MockMonitoringController)
		mockController.On("ValidateMonitoring", monitoring.Config).Return(nil)
		mockControllers.On("Get", monitoring.Type).Return(mockController, nil)

		result, err := registry.Register(monitoring.Type, monitoring)
		assert.NoError(t, err)
		assert.Equal(t, id, result.ID)
	})

	t.Run("validation error", func(t *testing.T) {
		registry, mockControllers := setupTestRegistry(t)
		monitoring := &model.Monitoring{Type: "prometheus"}
		expectedErr := errors.New("validation error")

		mockController := new(MockMonitoringController)
		mockController.On("ValidateMonitoring", monitoring.Config).Return(expectedErr)
		mockControllers.On("Get", monitoring.Type).Return(mockController, nil)

		_, err := registry.Register(monitoring.Type, monitoring)
		assert.EqualError(t, err, expectedErr.Error())
	})

	t.Run("controller not found", func(t *testing.T) {
		registry, mockControllers := setupTestRegistry(t)
		monitoring := &model.Monitoring{Type: "unknown"}
		expectedErr := errors.New("controller not found")

		// Мок возвращает nil для контроллера и ошибку
		mockControllers.On("Get", monitoring.Type).Return(nil, expectedErr)

		_, err := registry.Register(monitoring.Type, monitoring)
		assert.EqualError(t, err, expectedErr.Error())
		mockControllers.AssertExpectations(t)
	})

	t.Run("controller is nil", func(t *testing.T) {
		registry, mockControllers := setupTestRegistry(t)
		monitoring := &model.Monitoring{Type: "nil-controller"}

		// Мок возвращает nil контроллер без ошибки
		mockControllers.On("Get", monitoring.Type).Return(nil, nil)

		_, err := registry.Register(monitoring.Type, monitoring)
		assert.EqualError(t, err, "controller is 'nil'!")
		mockControllers.AssertExpectations(t)
	})

	t.Run("duplicate registration", func(t *testing.T) {
		registry, mockControllers := setupTestRegistry(t)
		id := uuid.New().String()
		monitoring1 := &model.Monitoring{ID: id, Type: "prometheus"}
		monitoring2 := &model.Monitoring{ID: id, Type: "prometheus"}

		// Настраиваем мок контроллер
		mockController := new(MockMonitoringController)
		mockController.On("ValidateMonitoring", mock.Anything).Return(nil).Twice()

		// Ожидаем два вызова Get() - по одному для каждой регистрации
		mockControllers.On("Get", monitoring1.Type).Return(mockController, nil).Twice()

		// Первая регистрация
		_, err := registry.Register(monitoring1.Type, monitoring1)
		assert.NoError(t, err)

		// Вторая регистрация с тем же ID
		_, err = registry.Register(monitoring2.Type, monitoring2)
		assert.EqualError(t, err, "monitoring system already registered")

		// Проверяем, что все ожидания выполнены
		mockController.AssertExpectations(t)
		mockControllers.AssertExpectations(t)
	})
}

func TestGet(t *testing.T) {
	t.Run("existing monitoring", func(t *testing.T) {
		registry, mockControllers := setupTestRegistry(t)
		monitoring := &model.Monitoring{Type: "prometheus"}

		mockController := new(MockMonitoringController)
		mockController.On("ValidateMonitoring", monitoring.Config).Return(nil)
		mockControllers.On("Get", monitoring.Type).Return(mockController, nil)

		registered, err := registry.Register(monitoring.Type, monitoring)
		assert.NoError(t, err)

		result, err := registry.Get(registered.ID)
		assert.NoError(t, err)
		assert.Equal(t, registered.ID, result.ID)
	})

	t.Run("non-existing monitoring", func(t *testing.T) {
		registry, _ := setupTestRegistry(t)
		_, err := registry.Get("non-existing-id")
		assert.EqualError(t, err, "monitoring system not found")
	})

	t.Run("empty ID", func(t *testing.T) {
		registry, _ := setupTestRegistry(t)
		_, err := registry.Get("")
		assert.EqualError(t, err, "monitoring system not found")
	})
}

func TestUpdate(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		registry, mockControllers := setupTestRegistry(t)
		original := &model.Monitoring{Type: "prometheus", Config: map[string]string{"url": "old"}}

		// Регистрация
		mockController := new(MockMonitoringController)
		mockController.On("ValidateMonitoring", original.Config).Return(nil)
		mockControllers.On("Get", original.Type).Return(mockController, nil)

		registered, err := registry.Register(original.Type, original)
		assert.NoError(t, err)

		// Обновление
		updated := &model.Monitoring{
			Type:   "prometheus",
			Config: map[string]string{"url": "new"},
		}

		mockController.On("ValidateMonitoring", updated.Config).Return(nil)

		err = registry.Update(registered.ID, updated)
		assert.NoError(t, err)

		// Проверка
		result, err := registry.Get(registered.ID)
		assert.NoError(t, err)
		assert.Equal(t, "new", result.Config["url"])
		assert.Equal(t, registered.ID, result.ID) // ID не должен измениться
		assert.NotNil(t, result.EventHistory)     // История событий сохраняется
		assert.NotNil(t, result.StatusHistory)    // История статусов сохраняется
	})

	t.Run("update non-existing monitoring", func(t *testing.T) {
		registry, _ := setupTestRegistry(t)
		err := registry.Update("non-existing-id", &model.Monitoring{})
		assert.EqualError(t, err, "monitoring system not found")
	})

	t.Run("validation error on update", func(t *testing.T) {
		registry, mockControllers := setupTestRegistry(t)
		original := &model.Monitoring{Type: "prometheus"}

		// Регистрация
		mockController := new(MockMonitoringController)
		mockController.On("ValidateMonitoring", original.Config).Return(nil)
		mockControllers.On("Get", original.Type).Return(mockController, nil)

		registered, err := registry.Register(original.Type, original)
		assert.NoError(t, err)

		// Обновление с ошибкой
		updated := &model.Monitoring{Type: "prometheus", Config: map[string]string{"invalid": "config"}}
		expectedErr := errors.New("validation error")

		mockController.On("ValidateMonitoring", updated.Config).Return(expectedErr)

		err = registry.Update(registered.ID, updated)
		//assert.EqualError(t, err, nil)
	})
}

func TestDelete(t *testing.T) {
	t.Run("successful deletion", func(t *testing.T) {
		registry, mockControllers := setupTestRegistry(t)
		monitoring := &model.Monitoring{Type: "prometheus"}

		mockController := new(MockMonitoringController)
		mockController.On("ValidateMonitoring", monitoring.Config).Return(nil)
		mockControllers.On("Get", monitoring.Type).Return(mockController, nil)

		registered, err := registry.Register(monitoring.Type, monitoring)
		assert.NoError(t, err)

		err = registry.Delete(registered.ID)
		assert.NoError(t, err)

		_, err = registry.Get(registered.ID)
		assert.EqualError(t, err, "monitoring system not found")
	})

	t.Run("delete non-existing monitoring", func(t *testing.T) {
		registry, _ := setupTestRegistry(t)
		err := registry.Delete("non-existing-id")
		assert.EqualError(t, err, "monitoring system not found")
	})
}

func TestList(t *testing.T) {
	t.Run("empty registry", func(t *testing.T) {
		registry, _ := setupTestRegistry(t)
		list, err := registry.List()
		assert.NoError(t, err)
		assert.Empty(t, list)
	})

	t.Run("registry with multiple monitorings", func(t *testing.T) {
		registry, mockControllers := setupTestRegistry(t)

		// Настройка моков
		mockController := new(MockMonitoringController)
		mockController.On("ValidateMonitoring", mock.Anything).Return(nil)
		mockControllers.On("Get", mock.Anything).Return(mockController, nil)

		// Регистрация нескольких мониторингов
		mon1 := &model.Monitoring{Type: "prometheus"}
		mon2 := &model.Monitoring{Type: "grafana"}

		_, err := registry.Register(mon1.Type, mon1)
		assert.NoError(t, err)
		_, err = registry.Register(mon2.Type, mon2)
		assert.NoError(t, err)

		// Проверка списка
		list, err := registry.List()
		assert.NoError(t, err)
		assert.Len(t, list, 2)

		// Проверка, что ID уникальны
		assert.NotEqual(t, list[0].ID, list[1].ID)
	})
}

func TestCheckConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		registry, mockControllers := setupTestRegistry(t)
		config := map[string]string{"url": "http://example.com"}

		mockController := new(MockMonitoringController)
		mockController.On("ValidateMonitoring", config).Return(nil)
		mockControllers.On("Get", "prometheus").Return(mockController, nil)

		err := registry.CheckConfig("prometheus", config)
		assert.NoError(t, err)
	})

	t.Run("invalid config", func(t *testing.T) {
		registry, mockControllers := setupTestRegistry(t)
		config := map[string]string{"url": "invalid"}
		expectedErr := errors.New("invalid url")

		mockController := new(MockMonitoringController)
		mockController.On("ValidateMonitoring", config).Return(expectedErr)
		mockControllers.On("Get", "prometheus").Return(mockController, nil)

		err := registry.CheckConfig("prometheus", config)
		assert.EqualError(t, err, expectedErr.Error())
	})

	t.Run("controller not found", func(t *testing.T) {
		registry, mockControllers := setupTestRegistry(t)
		expectedErr := errors.New("controller not found")

		mockControllers.On("Get", "unknown").Return(nil, expectedErr)

		err := registry.CheckConfig("unknown", map[string]string{})
		assert.EqualError(t, err, expectedErr.Error())
	})
}
