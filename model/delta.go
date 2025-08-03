package model

import "time"

type DeltaType int

const (
	DeltaTypeCreate DeltaType = iota
	DeltaTypeUpdate
	DeltaTypeDelete
)

type Delta struct {
	ID        string
	Type      DeltaType
	State     map[string]interface{}
	FieldMask []string
	Timestamp time.Time
	Causality []string // Для отслеживания причинно-следственных связей
}
