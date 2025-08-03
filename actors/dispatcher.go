package actors

import "github.com/sirupsen/logrus"

// Dispatcher управляет выполнением акторов
type Dispatcher struct {
	workers   int
	workQueue chan func()
	logger    *logrus.Logger
}

func NewDispatcher(workers int, logger *logrus.Logger) *Dispatcher {
	d := &Dispatcher{
		workers:   workers,
		workQueue: make(chan func(), 10000),
		logger:    logger,
	}
	d.start()
	return d
}

func (d *Dispatcher) start() {
	for i := 0; i < d.workers; i++ {
		go d.worker()
	}
}

func (d *Dispatcher) worker() {
	for task := range d.workQueue {
		func() {
			defer func() {
				if r := recover(); r != nil {
					d.logger.Errorf("actor panic: %v", r)
				}
			}()
			task()
		}()
	}
}

func (d *Dispatcher) Schedule(fn func()) {
	select {
	case d.workQueue <- fn:
	default:
		d.logger.Warn("dispatcher queue overflow")
	}
}

func (d *Dispatcher) SetWorkers(num int) {
	d.workers = num

}
