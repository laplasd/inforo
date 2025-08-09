package actors_test

import (
	"testing"
	"time"

	"github.com/laplasd/inforo/actors"
	"github.com/laplasd/inforo/actorsystem"
	"github.com/laplasd/inforo/hlc"
	"github.com/laplasd/inforo/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockCRDTStore - мок хранилища дельт
type MockCRDTStore struct {
	mock.Mock
}

func (m *MockCRDTStore) Merge(delta model.Delta) (bool, error) {
	args := m.Called(delta)
	return args.Bool(0), args.Error(1)
}

func (m *MockCRDTStore) GetAll(id string) ([]model.Delta, error) {
	args := m.Called(id)
	return args.Get(0).([]model.Delta), args.Error(1)
}

// TestComponentActor_Create тестирует создание компонента
func TestComponentActor_Create(t *testing.T) {
	logger := logrus.New()
	store := new(MockCRDTStore)
	hlc := hlc.NewHLC()

	// Настраиваем мок хранилища
	testComp := &model.Component{
		ID:   "test-1",
		Type: "test-type",
	}
	expectedDelta := model.Delta{
		ID:        testComp.ID,
		Type:      model.DeltaTypeCreate,
		State:     map[string]interface{}{"component": *testComp},
		Timestamp: hlc.Now(),
	}
	store.On("Merge", expectedDelta).Return(true, nil)

	// Создаем актор с моком хранилища
	actor := actors.NewComponentActor(logger)
	actor.Store = store
	actor.HLC = hlc

	// Тестовый контекст
	ctx := actorsystem.ActorContext{
		sender: actor,
	}

	t.Run("successful creation", func(t *testing.T) {
		respChan := make(chan interface{}, 1)
		msg := actorsystem.ComponentRegisterMsg{
			Component: *testComp,
			ReplyTo:   respChan,
		}

		// Отправляем сообщение
		actor.Receive(ctx, &actorsystem.AskMessage{
			Payload:  msg,
			Response: respChan,
		})

		// Проверяем ответ
		select {
		case resp := <-respChan:
			registerResp, ok := resp.(actorsystem.ComponentRegisterResponse)
			require.True(t, ok, "unexpected response type")
			assert.Equal(t, testComp.ID, registerResp.Component.ID)
			assert.Nil(t, registerResp.Error)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for response")
		}
	})

	t.Run("store error", func(t *testing.T) {
		store.On("Merge", expectedDelta).Return(false, assert.AnError).Once()
		respChan := make(chan interface{}, 1)
		msg := actorsystem.ComponentRegisterMsg{
			Component: *testComp,
			ReplyTo:   respChan,
		}

		actor.Receive(ctx, &actorsystem.AskMessage{
			Payload:  msg,
			Response: respChan,
		})

		select {
		case resp := <-respChan:
			registerResp := resp.(actorsystem.ComponentRegisterResponse)
			assert.NotNil(t, registerResp.Error)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for response")
		}
	})
}

// TestComponentActor_Read тестирует чтение компонента
func TestComponentActor_Read(t *testing.T) {
	logger := logrus.New()
	store := new(MockCRDTStore)
	hlc := hlc.NewHLC()

	// Тестовые данные
	testComp := &model.Component{
		ID:   "test-1",
		Name: "Test Component",
	}
	testDeltas := []model.Delta{
		{
			ID:        testComp.ID,
			Type:      model.DeltaTypeCreate,
			State:     map[string]interface{}{"component": *testComp},
			Timestamp: hlc.Now(),
		},
	}

	actor := actors.NewComponentActor(logger)
	actor.Store = store
	actor.HLC = hlc

	ctx := actorsystem.ActorContext{
		Self: actor,
	}

	t.Run("successful read", func(t *testing.T) {
		store.On("GetAll", testComp.ID).Return(testDeltas, nil).Once()

		respChan := make(chan interface{}, 1)
		msg := actorsystem.ComponentReadMsg{
			ComponentID: testComp.ID,
			ReplyTo:     respChan,
		}

		actor.Receive(ctx, &actorsystem.AskMessage{
			Payload:  msg,
			Response: respChan,
		})

		select {
		case resp := <-respChan:
			readResp := resp.(actorsystem.ComponentReadResponse)
			assert.Equal(t, testComp.ID, readResp.Component.ID)
			assert.Equal(t, testComp.Name, readResp.Component.Name)
			assert.Nil(t, readResp.Error)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for response")
		}
	})

	t.Run("not found", func(t *testing.T) {
		store.On("GetAll", "not-exist").Return([]model.Delta{}, nil).Once()

		respChan := make(chan interface{}, 1)
		msg := actorsystem.ComponentReadMsg{
			ComponentID: "not-exist",
			ReplyTo:     respChan,
		}

		actor.Receive(ctx, &actorsystem.AskMessage{
			Payload:  msg,
			Response: respChan,
		})

		select {
		case resp := <-respChan:
			readResp := resp.(actorsystem.ComponentReadResponse)
			assert.Nil(t, readResp.Component)
			assert.NotNil(t, readResp.Error)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for response")
		}
	})

	t.Run("store error", func(t *testing.T) {
		store.On("GetAll", testComp.ID).Return([]model.Delta{}, assert.AnError).Once()

		respChan := make(chan interface{}, 1)
		msg := actorsystem.ComponentReadMsg{
			ComponentID: testComp.ID,
			ReplyTo:     respChan,
		}

		actor.Receive(ctx, &actorsystem.AskMessage{
			Payload:  msg,
			Response: respChan,
		})

		select {
		case resp := <-respChan:
			readResp := resp.(actorsystem.ComponentReadResponse)
			assert.Nil(t, readResp.Component)
			assert.NotNil(t, readResp.Error)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for response")
		}
	})
}

// TestApplyDelta тестирует применение дельт к компоненту
func TestApplyDelta(t *testing.T) {
	logger := logrus.New()
	actor := actors.NewComponentActor(logger)

	baseComp := &model.Component{
		ID:   "test-1",
		Name: "Initial Name",
		Metadata: map[string]string{
			"key1": "value1",
		},
	}

	t.Run("merge all fields", func(t *testing.T) {
		delta := model.Delta{
			State: map[string]interface{}{
				"Name":     "New Name",
				"Metadata": map[string]string{"key2": "value2"},
			},
		}

		err := actor.ApplyDelta(baseComp, delta)
		assert.Nil(t, err)
		assert.Equal(t, "New Name", baseComp.Name)
		assert.Equal(t, "value1", baseComp.Metadata["key1"]) // старое значение сохраняется
		assert.Equal(t, "value2", baseComp.Metadata["key2"]) // новое поле добавляется
	})

	t.Run("merge with field mask", func(t *testing.T) {
		delta := model.Delta{
			FieldMask: []string{"Name", "Metadata.key3"},
			State: map[string]interface{}{
				"Name":     "Another Name",
				"Metadata": map[string]string{"key3": "value3"},
			},
		}

		err := actor.ApplyDelta(baseComp, delta)
		assert.Nil(t, err)
		assert.Equal(t, "Another Name", baseComp.Name)
		assert.Equal(t, "value3", baseComp.Metadata["key3"])
		assert.Nil(t, baseComp.Metadata["key2"]) // не должно измениться, т.к. не в field mask
	})
}

// TestReconstructComponent тестирует реконструкцию компонента из дельт
func TestReconstructComponent(t *testing.T) {
	logger := logrus.New()
	store := new(MockCRDTStore)
	actor := actors.NewComponentActor(logger)
	actor.Store = store

	compID := "test-1"
	deltas := []model.Delta{
		{
			ID:   compID,
			Type: model.DeltaTypeCreate,
			State: map[string]interface{}{
				"Name": "Initial Name",
			},
		},
		{
			ID:   compID,
			Type: model.DeltaTypeUpdate,
			State: map[string]interface{}{
				"Name": "Updated Name",
			},
		},
	}

	store.On("GetAll", compID).Return(deltas, nil).Once()

	comp, err := actor.ReconstructComponent(compID)
	assert.Nil(t, err)
	assert.Equal(t, "Updated Name", comp.Name)
}

// TestUnknownMessageType тестирует обработку неизвестных типов сообщений
func TestUnknownMessageType(t *testing.T) {
	logger := logrus.New()
	actor := actors.NewComponentActor(logger)

	ctx := actorsystem.ActorContext{
		Self: actor,
	}

	// Неизвестный тип сообщения
	unknownMsg := struct {
		Field string
	}{
		Field: "test",
	}

	// Проверяем, что актор не паникует
	assert.NotPanics(t, func() {
		actor.Receive(ctx, &actorsystem.AskMessage{
			Payload:  unknownMsg,
			Response: make(chan interface{}, 1),
		})
	})
}
