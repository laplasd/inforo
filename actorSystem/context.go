package actorsystem

import "context"

// ActorContext предоставляет контекст для обработки сообщений
type ActorContext struct {
	context.Context
	selfPath  string
	sender    ActorRef
	system    *ActorSystem
	replyChan chan<- interface{} // Канал для ответа
}

func NewActorContext(parent context.Context, path string, system *ActorSystem) ActorContext {
	return ActorContext{
		Context:  parent,
		selfPath: path,
		system:   system,
	}
}

func (c ActorContext) Self() ActorRef {
	return c.system.GetActor(c.selfPath)
}

func (c ActorContext) Sender() ActorRef {
	return c.sender
}

func (c ActorContext) System() *ActorSystem {
	return c.system
}

func (ctx *ActorContext) Reply(response interface{}) {
	if ctx.replyChan != nil {
		ctx.replyChan <- response
	}
}

// askMessage - внутреннее сообщение для Ask-паттерна
type AskMessage struct {
	Payload  interface{}
	Response chan<- interface{}
}
