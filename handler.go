package fswatch

import (
	"sync/atomic"
	"time"

	"go.shu.run/log"
)

type Handler interface {
	Name() string
	Delay() time.Duration
	Match(e Event) bool
	Run() error
	Stop() error
}

func NewFunc(handler Handler) *Runner {
	return &Runner{
		Handler: handler,
		waiting: 0,
	}
}

type Runner struct {
	Handler
	waiting int64
}

func (h *Runner) Execute() {
	if h.waiting == 1 {
		log.Debugf("执行[%s]等待 -> 跳过", h.Name())
		return
	}
	atomic.StoreInt64(&h.waiting, 1)
	time.AfterFunc(h.Delay(), func() {
		defer h.recover()
		atomic.StoreInt64(&h.waiting, 0)
		if err := h.Run(); err != nil {
			log.Errorf("执行[%s]出错 -> %v", h.Name(), err)
		}
	})
}

func (h *Runner) recover() {
	if re := recover(); re != nil {
		log.Errorf("%v", re)
	}
}
