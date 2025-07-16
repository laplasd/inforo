package api

type ControllerRegistry interface {
	Register(componentType string, controller Controller) error
	Get(componentType string) (Controller, error)
	Update(id string, comp Controller) error
	Delete(id string) error
	List() ([]Controller, error)
	ListType() ([]string, error)
}

type Controller interface {
	RunTask(TaskMeta map[string]string, ComponentMeta map[string]string) error
	ValideTask(TaskMeta map[string]string) error
	ValideComponent(ComponentMeta map[string]string) error
	CheckComponent(ComponentMeta map[string]string) error
}
