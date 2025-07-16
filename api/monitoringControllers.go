package api

type MonitoringControllerRegistry interface {
	Register(monitorType string, controller MonitoringController) error
	Get(componentType string) (MonitoringController, error)
	Update(id string, comp MonitoringController) error
	Delete(id string) error
	List() ([]MonitoringController, error)
}

type MonitoringController interface {
	RunCheck(monitorMeta map[string]string) error
	CheckMonitoring(config map[string]string) error
	ValidateCheck(monitorMeta map[string]string) error
	ValidateMonitoring(config map[string]string) error
}
