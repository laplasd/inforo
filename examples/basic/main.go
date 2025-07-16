// Базовое использование inforo Core
package main

import (
	"fmt"
	"log"

	"github.com/laplasd/inforo"
	"github.com/laplasd/inforo/model"
)

func main() {
	// 1. Инициализация Core с настройками по умолчанию
	core := inforo.NewDefaultCore()

	// 2. Создание и регистрация компонента
	component := model.Component{
		ID:      "web-server",
		Name:    "NGINX",
		Type:    "webserver",
		Version: "1.23.0",
	}

	registeredComp, err := core.Components.Register(component)
	if err != nil {
		log.Fatalf("Failed to register component: %v", err)
	}

	fmt.Printf("Registered component: %+v\n", registeredComp)

	// 3. Создание простой задачи
	task := model.Task{
		ID:         "deploy-web",
		Name:       "Deploy Web Server",
		Type:       model.UpdateTask,
		Components: []string{"web-server"},
	}

	fmt.Printf("Created task: %+v\n", task)
}
