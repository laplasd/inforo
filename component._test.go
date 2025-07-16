package inforo_test

import (
	"testing"

	"github.com/laplasd/inforo"

	"github.com/laplasd/inforo/model"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock controller ---
type mockController struct {
	validateErr error
}

func (m *mockController) ValideComponent(metadata map[string]string) error {
	return m.validateErr
}

func (m *mockController) CheckComponent(metadata map[string]string) error {
	return nil
}

func (m *mockController) ValideTask(r map[string]string) error {
	return nil
}

func (m *mockController) RunTask(r map[string]string, p map[string]string) error {
	return nil
}

// --- helper ---
func newTestCore() *inforo.Core {
	logger := logrus.New()
	opts := inforo.CoreOptions{
		Logger: logger,
	}
	c := inforo.NewCore(opts)
	c.Controllers.Register("mock", &mockController{})
	return c
}

// --- tests ---
func TestRegisterComponent_Success(t *testing.T) {
	c := newTestCore()
	comp := model.Component{ID: "c1", Name: "Comp1", Type: "mock", Version: "1.0.0", Metadata: map[string]string{}}

	registered, err := c.Components.Register(comp)
	assert.NoError(t, err)
	assert.Equal(t, "c1", registered.ID)
	assert.Equal(t, model.StatusPending, registered.StatusHistory.LastStatus)
}

func TestRegisterComponent_AlreadyRegistered(t *testing.T) {
	c := newTestCore()
	comp := model.Component{ID: "c1", Type: "mock", Version: "1.0.0", Metadata: map[string]string{}}
	_, _ = c.Components.Register(comp)
	_, err := c.Components.Register(comp)
	assert.Error(t, err)
}

func TestRegisterComponent_UnsupportedType(t *testing.T) {
	c := newTestCore()
	comp := model.Component{ID: "bad", Type: "unknown", Version: "1.0.0", Metadata: map[string]string{}}
	_, err := c.Components.Register(comp)
	assert.Error(t, err)
}

func TestGetComponent_Success(t *testing.T) {
	c := newTestCore()
	comp := model.Component{ID: "get1", Type: "mock", Version: "1.0.0", Metadata: map[string]string{}}
	got, err := c.Components.Register(comp)

	assert.NoError(t, err)
	assert.Equal(t, "get1", got.ID)
}

func TestGetComponent_NotFound(t *testing.T) {
	c := newTestCore()
	_, err := c.Components.Get("unknown")
	assert.Error(t, err)
}

func TestUpdateComponent_Success(t *testing.T) {
	c := newTestCore()
	orig := model.Component{ID: "upd1", Name: "Old", Type: "mock", Version: "1.0.0", Metadata: map[string]string{}}
	_, _ = c.Components.Register(orig)

	updated := &model.Component{ID: "upd1", Name: "New", Type: "mock", Version: "1.0.0", Metadata: map[string]string{"foo": "bar"}}
	err := c.Components.Update("upd1", updated)
	res, _ := c.Components.Get(updated.ID)
	assert.NoError(t, err)
	assert.Equal(t, "New", res.Name)
	assert.Equal(t, "bar", res.Metadata["foo"])
}

func TestUpdateComponent_NotFound(t *testing.T) {
	c := newTestCore()
	comp := &model.Component{ID: "x", Type: "mock", Version: "1.0.0", Metadata: map[string]string{}}
	err := c.Components.Update("x", comp)
	assert.Error(t, err)
}

func TestUpdateComponent_InvalidType(t *testing.T) {
	c := newTestCore()
	comp := &model.Component{ID: "x", Type: "mock", Version: "1.0.0", Metadata: map[string]string{}}
	_, _ = c.Components.Register(*comp)

	comp.Type = "unknown"
	err := c.Components.Update("x", comp)
	assert.Error(t, err)
}

func TestDeleteComponent_Success(t *testing.T) {
	c := newTestCore()
	comp := model.Component{ID: "del1", Type: "mock", Version: "1.0.0", Metadata: map[string]string{}}
	_, _ = c.Components.Register(comp)

	err := c.Components.Delete("del1")
	assert.NoError(t, err)
}

func TestDeleteComponent_NotFound(t *testing.T) {
	c := newTestCore()
	err := c.Components.Delete("nope")
	assert.Error(t, err)
}

func TestListComponents(t *testing.T) {
	c := newTestCore()
	_, _ = c.Components.Register(model.Component{ID: "a", Type: "mock", Version: "1.0.0", Metadata: map[string]string{}})
	_, _ = c.Components.Register(model.Component{ID: "b", Type: "mock", Version: "1.0.0", Metadata: map[string]string{}})

	list, _ := c.Components.List()
	assert.Len(t, list, 2)
}

func TestComponentRegistry_Register_CreatesEventHistory(t *testing.T) {
	c := newTestCore()

	// Убедимся, что контроллер для типа "mock" существует
	//err := c.Controllers.Register("mock", &MockController{})
	//require.NoError(t, err)

	testComp := model.Component{
		Name:     "test-component",
		Version:  "1.0.0",
		Type:     "mock",
		Metadata: map[string]string{},
	}

	registeredComp, err := c.Components.Register(testComp)
	require.NoError(t, err)
	require.NotNil(t, registeredComp)

	// Проверяем что EventHistory был создан
	require.NotNil(t, registeredComp.EventHistory, "EventHistory should be initialized")
	require.NotNil(t, registeredComp.EventHistory.Event, "Events slice should be initialized")

	// Проверяем добавленное событие
	assert.GreaterOrEqual(t, len(registeredComp.EventHistory.Event), 1, "Should have at least one event")
	assert.Equal(t, "Created component!", registeredComp.EventHistory.Event[0].Message)
}
