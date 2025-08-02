package actors

// ActorContext предоставляет контекст для обработки сообщений
type ActorContext struct {
	selfPath string
	sender   ActorRef
	system   *ActorSystem
}

func NewActorContext(path string, system *ActorSystem) ActorContext {
	return ActorContext{
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

// askMessage - внутреннее сообщение для Ask-паттерна
type askMessage struct {
	payload  interface{}
	response chan<- interface{}
}
