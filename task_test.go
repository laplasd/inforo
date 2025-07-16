package inforo_test

import (
	"laplasd/internal/core"
	"laplasd/pkg/kernel"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setupCoreWithComponent() *core.Core {
	logger := logrus.New()
	c := core.New(logger)

	// Зарегистрируем один компонент
	c.RegisterComponent(kernel.Component{
		ID:   "component-1",
		Name: "Test Component",
		Type: "kuber-controller",
	})

	return c
}

func TestRegisterTask_Success(t *testing.T) {
	c := setupCoreWithComponent()

	task := &kernel.Task{
		ID:          "task-1",
		Name:        "Test Task",
		Type:        "update",
		ComponentID: "component-1",
	}

	registered, err := c.RegisterTask(task)

	assert.NoError(t, err)
	assert.NotNil(t, registered)
	assert.Equal(t, kernel.StatusPending, registered.Status.Status)
}

func TestRegisterTask_Duplicate(t *testing.T) {
	c := setupCoreWithComponent()

	task := &kernel.Task{
		ID:          "task-1",
		Name:        "Test Task",
		Type:        "update",
		ComponentID: "component-1",
	}

	_, _ = c.RegisterTask(task)
	_, err := c.RegisterTask(task)

	assert.Error(t, err)
	assert.EqualError(t, err, "task already exists")
}

func TestRegisterTask_InvalidComponent(t *testing.T) {
	c := setupCoreWithComponent()

	task := &kernel.Task{
		ID:          "task-2",
		Name:        "Invalid",
		Type:        "update",
		ComponentID: "nonexistent",
	}

	_, err := c.RegisterTask(task)

	assert.Error(t, err)
	assert.EqualError(t, err, "component with given ComponentID not found")
}

func TestRegisterTask_InvalidType(t *testing.T) {
	c := setupCoreWithComponent()

	task := &kernel.Task{
		ID:          "task-3",
		Name:        "Bad Type",
		Type:        "invalid-type",
		ComponentID: "component-1",
	}

	_, err := c.RegisterTask(task)

	assert.Error(t, err)
	assert.EqualError(t, err, "invalid task type")
}

func TestGetTask(t *testing.T) {
	c := setupCoreWithComponent()

	task := &kernel.Task{
		ID:          "task-1",
		Name:        "Test Task",
		Type:        "update",
		ComponentID: "component-1",
	}

	_, _ = c.RegisterTask(task)

	fetched, err := c.GetTask("task-1")

	assert.NoError(t, err)
	assert.Equal(t, "task-1", fetched.ID)
}

func TestGetTask_NotFound(t *testing.T) {
	c := setupCoreWithComponent()

	_, err := c.GetTask("missing")
	assert.Error(t, err)
	assert.EqualError(t, err, "task not found")
}

func TestUpdateTask(t *testing.T) {
	c := setupCoreWithComponent()

	task := &kernel.Task{
		ID:          "task-1",
		Name:        "Original",
		Type:        "update",
		ComponentID: "component-1",
	}
	_, _ = c.RegisterTask(task)

	updated := &kernel.Task{
		ID:          "task-1",
		Name:        "Updated",
		Type:        "update",
		ComponentID: "component-1",
		//Status:      kernel.StatusSuccess,
	}
	err := c.UpdateTask("task-1", updated)

	assert.NoError(t, err)

	got, _ := c.GetTask("task-1")
	assert.Equal(t, "Updated", got.Name)
	assert.Equal(t, kernel.StatusPending, got.Status.Status)
}

func TestDeleteTask(t *testing.T) {
	c := setupCoreWithComponent()

	task := &kernel.Task{
		ID:          "task-1",
		Name:        "ToDelete",
		Type:        "update",
		ComponentID: "component-1",
	}
	_, _ = c.RegisterTask(task)

	err := c.DeleteTask("task-1")
	assert.NoError(t, err)

	_, err = c.GetTask("task-1")
	assert.Error(t, err)
}

func TestListTasks(t *testing.T) {
	c := setupCoreWithComponent()

	tasks := []*kernel.Task{
		{ID: "task-1", Name: "Task 1", Type: "update", ComponentID: "component-1"},
		{ID: "task-2", Name: "Task 2", Type: "check", ComponentID: "component-1"},
	}

	for _, tsk := range tasks {
		_, _ = c.RegisterTask(tsk)
	}

	list := c.ListTasks()

	assert.Len(t, list, 2)
}
