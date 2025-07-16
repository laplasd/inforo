package api

import "github.com/laplasd/inforo/model"

type MonitoringRegistry interface {
	StatusProvider
	Register(id string, comp model.Monitoring) error
	Get(id string) (model.Monitoring, error)
	Update(id string, comp model.Monitoring) error
	Delete(id string) error
	List() ([]model.Monitoring, error)
}
