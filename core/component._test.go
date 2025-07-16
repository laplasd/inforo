package core_test

import (
	"testing"

	"github.com/laplasd/inforo/core"
	"github.com/laplasd/inforo/model"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
func newTestCore() *core.Core {
	logger := logrus.New()
	c := core.New(logger)
	c.RegisterController("mock", &mockController{})
	return c
}

// --- tests ---
func TestRegisterComponent_Success(t *testing.T) {
	c := newTestCore()
	comp := model.Component{ID: "c1", Name: "Comp1", Type: "kuber-controller", Metadata: map[string]string{}}

	registered, err := c.RegisterComponent(comp)
	assert.NoError(t, err)
	assert.Equal(t, "c1", registered.ID)
	assert.Equal(t, model.StatusPending, registered.Status.Status)
}

func TestRegisterComponent_AlreadyRegistered(t *testing.T) {
	c := newTestCore()
	comp := model.Component{ID: "c1", Type: "kuber-controller", Metadata: map[string]string{}}
	_ = c.Components.Register(comp)
	err := c.Components.Register(comp)
	assert.Error(t, err)
}

func TestRegisterComponent_UnsupportedType(t *testing.T) {
	c := newTestCore()
	comp := model.Component{ID: "bad", Type: "unknown", Metadata: map[string]string{}}
	err := c.Components.Register(comp)
	assert.Error(t, err)
}

func TestGetComponent_Success(t *testing.T) {
	c := newTestCore()
	comp := model.Component{ID: "get1", Type: "kuber-controller", Metadata: map[string]string{}}
	_ = c.Components.Register(comp)

	got, err := c.Components.Get("get1")
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
	orig := model.Component{ID: "upd1", Name: "Old", Type: "kuber-controller", Metadata: map[string]string{}}
	_ = c.Components.Register(orig)

	updated := model.Component{ID: "upd1", Name: "New", Type: "kuber-controller", Metadata: map[string]string{"foo": "bar"}}
	res, err := c.UpdateComponent("upd1", updated)
	assert.NoError(t, err)
	assert.Equal(t, "New", res.Name)
	assert.Equal(t, "bar", res.Metadata["foo"])
}

func TestUpdateComponent_NotFound(t *testing.T) {
	c := newTestCore()
	comp := model.Component{ID: "x", Type: "kuber-controller", Metadata: map[string]string{}}
	_, err := c.UpdateComponent("x", comp)
	assert.Error(t, err)
}

func TestUpdateComponent_InvalidType(t *testing.T) {
	c := newTestCore()
	comp := model.Component{ID: "x", Type: "kuber-controller", Metadata: map[string]string{}}
	_, _ = c.RegisterComponent(comp)

	comp.Type = "unknown"
	_, err := c.UpdateComponent("x", comp)
	assert.Error(t, err)
}

func TestDeleteComponent_Success(t *testing.T) {
	c := newTestCore()
	comp := model.Component{ID: "del1", Type: "kuber-controller", Metadata: map[string]string{}}
	_, _ = c.RegisterComponent(comp)

	err := c.DeleteComponent("del1")
	assert.NoError(t, err)
}

func TestDeleteComponent_NotFound(t *testing.T) {
	c := newTestCore()
	err := c.DeleteComponent("nope")
	assert.Error(t, err)
}

func TestListComponents(t *testing.T) {
	c := newTestCore()
	_, _ = c.RegisterComponent(model.Component{ID: "a", Type: "kuber-controller", Metadata: map[string]string{}})
	_, _ = c.RegisterComponent(model.Component{ID: "b", Type: "kuber-controller", Metadata: map[string]string{}})

	list := c.ListComponents()
	assert.Len(t, list, 2)
}
