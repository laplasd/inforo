package api

import (
	"github.com/laplasd/inforo/model"
)

type TaskRegistry interface {
	StatusProvider
	Validate(task *model.Task) error
	Register(task *model.Task) (*model.Task, error)
	Get(id string) (*model.Task, error)
	Update(id string, comp *model.Task) error
	Delete(id string) error
	List() ([]*model.Task, error)
	// Process methods
	ForkAsync(TaskID string, executionID string) (string, error)
	Fork(TaskID string, executionID string) (string, error)
	RollBackAsync(TaskID string, executionID string) (string, error)
	RollBack(TaskID string, executionID string) (string, error)
	Status(TaskID string) (string, error)
	Stop(TaskID string) error
	Pause(TaskID string) error
}
