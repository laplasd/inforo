package api

import "github.com/laplasd/inforo/model"

type ComponentRegistry interface {
	StatusProvider
	Register(comp model.Component) (*model.Component, error)
	Get(id string) (*model.Component, error)
	Update(id string, comp *model.Component) error
	Delete(id string) error
	//
	Disable(id string) error
	Enable(id string) error
	List() ([]*model.Component, error)
}
