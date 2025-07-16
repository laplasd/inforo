package model

import "sync"

type DepensType string

const (
	// Strict - зависимость должна быть успешно выполнена (жесткая зависимость)
	// Если зависимость не выполнена - ошибка
	Strict DepensType = "strict"

	// Ordered - зависимость должна быть выполнена первой (порядковая зависимость)
	// Планировщик должен сначала выполнить эту задачу
	Ordered DepensType = "ordered"

	// Advisory - мягкая зависимость (рекомендательная)
	// Желательно выполнить зависимость, но не обязательно
	Advisory DepensType = "advisory"

	// Blocking - блокирующая зависимость
	// Текущая задача не может выполняться, пока зависимость не завершится
	Blocking DepensType = "blocking"
)

type Depends struct {
	Type DepensType
	ID   string
}

type TaskType string

const (
	UpdateTask   TaskType = "update"
	RollbackTask TaskType = "rollback"
	CheckTask    TaskType = "check"
)

type Task struct {
	ID            string            `json:"ID"`
	Name          string            `json:"Name"`
	Type          TaskType          `json:"Type"`
	Components    []string          `json:"Components"`
	RollBack      *Rollback         `json:"RollBack,omitempty"`
	DependsOn     []Depends         `json:"DependsOn,omitempty"`
	PreChecks     []*Check          `json:"PreChecks,omitempty"`
	PostChecks    []*Check          `json:"PostChecks,omitempty"`
	StatusHistory *StatusHistory    `json:"StatusHistory,omitempty"`
	EventHistory  *EventHistory     `json:"EventHistory,omitempty"`
	Metadata      map[string]string `json:"MetaData"`
	MU            sync.RWMutex      `json:"-"`
}
