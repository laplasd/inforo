package inforo_test

import (
	"testing"

	"github.com/laplasd/inforo"
	"github.com/laplasd/inforo/model"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setupCoreWithComponent() *inforo.Core {
	logger := logrus.New()
	opts := inforo.CoreOptions{
		Logger: logger,
	}
	c := inforo.NewCore(opts)

	// Зарегистрируем один компонент
	c.Components.Register(model.Component{
		ID:   "component-1",
		Name: "Test Component",
		Type: "kuber-controller",
	})

	return c
}

func TestRegisterTask_Success(t *testing.T) {
	c := setupCoreWithComponent()

	task := (model.Task{
		ID:         "task-1",
		Name:       "Test Task",
		Type:       "update",
		Components: []string{"component-1"},
	})

	registered, err := c.Tasks.Register(task)

	assert.NoError(t, err)
	assert.NotNil(t, registered)
	assert.Equal(t, model.StatusPending, registered.StatusHistory.LastStatus)
}

func TestRegisterTask_Duplicate(t *testing.T) {
	c := setupCoreWithComponent()

	task := model.Task{
		ID:         "task-1",
		Name:       "Test Task",
		Type:       "update",
		Components: []string{"component-1"},
	}

	_, _ = c.Tasks.Register(task)
	_, err := c.Tasks.Register(task)

	assert.Error(t, err)
	assert.EqualError(t, err, "task already exists")
}

func TestRegisterTask_InvalidComponent(t *testing.T) {
	c := setupCoreWithComponent()

	task := model.Task{
		ID:         "task-2",
		Name:       "Invalid",
		Type:       "update",
		Components: []string{"nonexistent"},
	}

	_, err := c.Tasks.Register(task)

	assert.Error(t, err)
	assert.EqualError(t, err, "component with given ComponentID not found")
}

func TestRegisterTask_InvalidType(t *testing.T) {
	c := setupCoreWithComponent()

	task := model.Task{
		ID:         "task-3",
		Name:       "Bad Type",
		Type:       "invalid-type",
		Components: []string{"component-1"},
	}

	_, err := c.Tasks.Register(task)

	assert.Error(t, err)
	assert.EqualError(t, err, "invalid task type")
}

func TestGetTask(t *testing.T) {
	c := setupCoreWithComponent()

	task := model.Task{
		ID:         "task-1",
		Name:       "Test Task",
		Type:       "update",
		Components: []string{"component-1"},
	}

	_, _ = c.Tasks.Register(task)

	fetched, err := c.Tasks.Get("task-1")

	assert.NoError(t, err)
	assert.Equal(t, "task-1", fetched.ID)
}

func TestGetTask_NotFound(t *testing.T) {
	c := setupCoreWithComponent()

	_, err := c.Tasks.Get("missing")
	assert.Error(t, err)
	assert.EqualError(t, err, "task not found")
}

func TestUpdateTask(t *testing.T) {
	c := setupCoreWithComponent()

	task := model.Task{
		ID:         "task-1",
		Name:       "Original",
		Type:       "update",
		Components: []string{"component-1"},
	}
	_, _ = c.Tasks.Register(task)

	updated := model.Task{
		ID:         "task-1",
		Name:       "Updated",
		Type:       "update",
		Components: []string{"component-1"},
		//Status:      kernel.StatusSuccess,
	}
	err := c.Tasks.Update("task-1", updated)

	assert.NoError(t, err)

	got, _ := c.Tasks.Get("task-1")
	assert.Equal(t, "Updated", got.Name)
	assert.Equal(t, model.StatusPending, got.StatusHistory.LastStatus)
}

func TestDeleteTask(t *testing.T) {
	c := setupCoreWithComponent()

	task := model.Task{
		ID:         "task-1",
		Name:       "ToDelete",
		Type:       "update",
		Components: []string{"component-1"},
	}
	_, _ = c.Tasks.Register(task)

	err := c.Tasks.Delete("task-1")
	assert.NoError(t, err)

	_, err = c.Tasks.Get("task-1")
	assert.Error(t, err)
}

func TestListTasks(t *testing.T) {
	c := setupCoreWithComponent()

	tasks := []model.Task{
		{ID: "task-1", Name: "Task 1", Type: "update", Components: []string{"component-1"}},
		{ID: "task-2", Name: "Task 2", Type: "check", Components: []string{"component-1"}},
	}

	for _, tsk := range tasks {
		_, _ = c.Tasks.Register(tsk)
	}

	list, _ := c.Tasks.List()

	assert.Len(t, list, 2)
}
