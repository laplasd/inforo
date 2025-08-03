package actors

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	actorsystem "github.com/laplasd/inforo/actorSystem"
	"github.com/laplasd/inforo/crdt"
	"github.com/laplasd/inforo/hlc"
	"github.com/laplasd/inforo/statecache"

	"github.com/laplasd/inforo/model"
	"github.com/sirupsen/logrus"
)

type ComponentActor struct {
	*actorsystem.BaseActor
	logger      *logrus.Logger
	handlers    map[model.Status]func(interface{})
	store       crdt.CRDTStore
	deltaStream chan model.Delta
	stateCache  *statecache.StateCache
	hlc         *hlc.HLC
}

func NewComponentActor(logger *logrus.Logger) *ComponentActor {
	act := &ComponentActor{
		logger:   logger,
		handlers: make(map[model.Status]func(interface{})),
	}
	act.BaseActor = actorsystem.NewBaseActor("internal.component", act.Receive, nil, *logger)
	return act
}

// CRUD-методы через сообщения
func (co *ComponentActor) Receive(ctx actorsystem.ActorContext, msg interface{}) {
	switch cmd := msg.(type) {
	case actorsystem.ComponentRegisterMsg:
		co.handleCreate(cmd, ctx)
	case actorsystem.ComponentReadMsg:
		co.handleRead(cmd, ctx)
	case actorsystem.AskMessage:
		fmt.Printf("unknown type. AskMessage")
	default:
		fmt.Printf("unknown type. Skip")

	}
}

// Create
func (co *ComponentActor) handleCreate(cmd actorsystem.ComponentRegisterMsg, ctx actorsystem.ActorContext) {
	// Преобразуем Component в map[string]interface{} для Delta.State

	co.logger.Debug("ComponentActor.handleCreate")
	delta := model.Delta{
		ID:        cmd.Component.ID,
		Type:      model.DeltaTypeCreate,
		State:     make(map[string]interface{}), // Теперь правильный тип
		Timestamp: co.hlc.Now(),
	}
	delta.State["component"] = cmd.Component

	if _, err := co.store.Merge(delta); err != nil {
		ctx.Reply(actorsystem.ComponentRegisterResponse{Error: err})
		return
	}

	co.stateCache.Put(cmd.Component.ID, cmd.Component)
	ctx.Reply(actorsystem.ComponentRegisterResponse{Component: &cmd.Component})
	co.Tell(actorsystem.ComponentRegisterResponse{Component: &cmd.Component})
}

// Read
func (co *ComponentActor) handleRead(cmd actorsystem.ComponentReadMsg, ctx actorsystem.ActorContext) {
	// Проверяем кэш
	if cached, exists := co.stateCache.Get(cmd.ComponentID); exists {
		ctx.Reply(actorsystem.ComponentReadResponse{
			Component: cached.(*model.Component),
		})
		return
	}

	// Реконструируем из дельт
	component, err := co.reconstructComponent(cmd.ComponentID)
	if err != nil {
		ctx.Reply(actorsystem.ComponentReadResponse{Error: err})
		return
	}

	// Обновляем кэш
	co.stateCache.Put(cmd.ComponentID, component)

	ctx.Reply(actorsystem.ComponentReadResponse{
		Component: component,
	})
}

func (co *ComponentActor) reconstructComponent(id string) (*model.Component, error) {
	// Получаем все дельты для компонента
	deltas, err := co.store.GetAll(id)
	if err != nil {
		return nil, err
	}

	// Сортируем дельты по времени
	sort.Slice(deltas, func(i, j int) bool {
		return deltas[i].Timestamp.Before(deltas[j].Timestamp)
	})

	// Инициализируем базовый компонент
	component := &model.Component{
		Metadata: make(map[string]string),
	}

	// Применяем дельты последовательно
	for _, delta := range deltas {
		if delta.Type == model.DeltaTypeDelete {
			return nil, fmt.Errorf("component %s was deleted", id)
		}

		if err := co.applyDelta(component, delta); err != nil {
			return nil, err
		}
	}

	return component, nil
}

func (co *ComponentActor) applyDelta(component *model.Component, delta model.Delta) error {
	component.MU.Lock()
	defer component.MU.Unlock()

	// Если FieldMask не указан, мерджим все поля
	if len(delta.FieldMask) == 0 {
		return co.mergeAllFields(component, delta.State)
	}

	// Мерджим только указанные поля
	return co.mergeSelectedFields(component, delta.State, delta.FieldMask)
}

func (co *ComponentActor) mergeAllFields(component *model.Component, state map[string]interface{}) error {
	// Обработка простых полей
	if val, exists := state["Name"]; exists {
		if name, ok := val.(string); ok {
			component.Name = name
		}
	}

	// Обработка вложенных структур
	if val, exists := state["StatusHistory"]; exists {
		if history, err := co.parseStatusHistory(val); err == nil {
			component.StatusHistory = history
		}
	}

	// Обработка метаданных
	if val, exists := state["Metadata"]; exists {
		if meta, ok := val.(map[string]string); ok {
			for k, v := range meta {
				component.Metadata[k] = v
			}
		}
	}

	return nil
}

func (co *ComponentActor) mergeSelectedFields(component *model.Component, state map[string]interface{}, fieldMask []string) error {
	component.MU.Lock()
	defer component.MU.Unlock()

	for _, fieldPath := range fieldMask {
		parts := strings.Split(fieldPath, ".")
		if len(parts) == 0 {
			continue
		}

		var current interface{} = state
		var found bool

		// Находим значение в state по пути fieldPath
		for _, part := range parts {
			if m, ok := current.(map[string]interface{}); ok {
				current, found = m[part]
				if !found {
					break
				}
			} else {
				found = false
				break
			}
		}

		if !found {
			co.logger.Warnf("field %s not found in delta state", fieldPath)
			continue
		}

		// Применяем изменение к конкретному полю
		switch parts[0] {
		case "Name":
			if val, ok := current.(string); ok && len(parts) == 1 {
				component.Name = val
			}
		case "Type":
			if val, ok := current.(string); ok && len(parts) == 1 {
				component.Type = val
			}
		case "Version":
			if val, ok := current.(string); ok && len(parts) == 1 {
				component.Version = val
			}
		case "StatusHistory":
			if len(parts) == 1 {
				if history, err := co.parseStatusHistory(current); err == nil {
					component.StatusHistory = history
				}
			} else {
				// Обработка вложенных полей StatusHistory
				co.mergeStatusHistoryField(component, parts[1:], current)
			}
		case "EventHistory":
			if len(parts) == 1 {
				if history, err := co.parseEventHistory(current); err == nil {
					component.EventHistory = history
				}
			}
		case "Metadata":
			if len(parts) == 2 {
				if component.Metadata == nil {
					component.Metadata = make(map[string]string)
				}
				if val, ok := current.(string); ok {
					component.Metadata[parts[1]] = val
				}
			}
		}
	}
	return nil
}

func (co *ComponentActor) mergeStatusHistoryField(component *model.Component, path []string, value interface{}) {
	if component.StatusHistory == nil {
		component.StatusHistory = &model.StatusHistory{}
	}

	// Пример обработки вложенного поля в StatusHistory
	if path[0] == "LastStatus" {
		if status, ok := value.(model.Status); ok {
			component.StatusHistory.LastStatus = status
		}
	}
	// Можно добавить обработку других полей StatusHistory
}

func (co *ComponentActor) parseStatusHistory(data interface{}) (*model.StatusHistory, error) {
	// Реальная реализация будет зависеть от структуры StatusHistory
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var history model.StatusHistory
	if err := json.Unmarshal(jsonData, &history); err != nil {
		return nil, err
	}

	return &history, nil
}

func (co *ComponentActor) parseEventHistory(data interface{}) (*model.EventHistory, error) {
	// Аналогично parseStatusHistory
	// ...

	return nil, nil
}
