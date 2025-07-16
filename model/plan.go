package model

import (
	"sync"
	"time"
)

type Plan struct {
	ID            string                `json:"id"` // Уникальный идентификатор плана
	TaskGraphs    []*TaskGraph          // Набор независимых графов задач
	RollbackStack []*RollbackCheckpoint // Стек точек отката
	StatusHistory *StatusHistory        `json:"StatusHistory,omitempty"` // История статусов плана
	EventHistory  *EventHistory         `json:"EventHistory,omitempty"`
	MU            sync.RWMutex          `json:"-"`
}

// TaskGraph представляет направленный ациклический граф задач
type TaskGraph struct {
	RootTaskID   string              // Корневая задача графа (если есть)
	Tasks        map[string]*Task    // Все задачи графа
	Dependencies map[string][]string // Прямые зависимости (task → dependsOn)
	Dependents   map[string][]string // Обратные зависимости (task ← requiredBy)
}

// RollbackCheckpoint содержит состояние для отката
type RollbackCheckpoint struct {
	GraphID   string                 // ID графа
	TaskID    string                 // ID задачи
	State     map[string]interface{} // Состояние компонентов
	Timestamp time.Time              // Время создания точки отката
}
