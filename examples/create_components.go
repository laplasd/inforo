package main

import (
	"fmt"

	"github.com/laplasd/inforo"
	"github.com/laplasd/inforo/model"
)

func main() {
	core := inforo.NewDefaultCore()
	component := model.Component{
		ID:       "",
		Name:     "",
		Type:     "",
		Version:  "",
		Metadata: make(map[string]string),
	}
	comp, err := core.Components.Register(component)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println("%s", comp)
}
