package api

import (
	"github.com/laplasd/inforo/model"
)

type PlanRegistry interface {
	StatusProvider
	// CRUD methods
	Register(tasks []model.Task) (*model.Plan, error)
	Get(id string) (*model.Plan, error)
	Update(id string, comp model.Plan) error
	Delete(id string) error
	List() ([]*model.Plan, error)
	// Process methods
	RunAsync(planID string, executionID string) (string, error)
	Run(planID string, executionID string) (string, error)
	Status(planID string) (model.Status, error)
	Stop(planID string) error
	Pause(planID string) error
}
