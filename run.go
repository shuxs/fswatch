package fswatch

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"go.shu.run/log"
)

func Start(path string) *Watcher {
	w := New(path)
	w.Start(context.Background())
	return w
}

func New(path string) *Watcher {
	f := &Watcher{}
	f.path = path
	f.nw, _ = fsnotify.NewWatcher()
	f.onError = func(err error) {
		log.Errorf("%v", err)
	}
	return f
}

//Watcher 监听执行
type Watcher struct {
	path   string
	nw     *fsnotify.Watcher
	cancel context.CancelFunc

	handlers []*Runner
	onError  func(err error)

	locker sync.Mutex
}

//HandleError 设置错误处理器
func (w *Watcher) HandleError(onError func(err error)) {
	w.onError = onError
}

//Add 添加监听目录
func (w *Watcher) Add(path string) (err error) {
	var file os.FileInfo
	if file, err = os.Stat(path); err != nil {
		return err
	}
	return w.add(filepath.Dir(path), file)
}

func (w *Watcher) Find(f func(hf *Runner) bool) bool {
	for _, hf := range w.handlers {
		if f(hf) {
			return true
		}
	}
	return false
}

//Handle 启动
func (w *Watcher) Handle(handler Handler) {
	w.locker.Lock()
	defer w.locker.Unlock()
	for _, h := range w.handlers {
		if h.Name() == handler.Name() {
			log.Infof("增加处理器: %s -> 重复", handler.Name())
			return
		}
	}
	log.Infof("增加处理器: %s", handler.Name())
	if hf, ok := handler.(*Runner); ok {
		w.handlers = append(w.handlers, hf)
	} else {
		w.handlers = append(w.handlers, NewFunc(handler))
	}
}

//Start 启动
func (w *Watcher) Start(ctx context.Context) {
	ctx, w.cancel = context.WithCancel(ctx)
	if err := w.Add(w.path); err != nil {
		w.onError(err)
		return
	}
	go w.selectEvent(ctx)
}

//Stop 停止
func (w *Watcher) Stop() {
	w.locker.Lock()
	defer w.locker.Unlock()
	if w.cancel != nil {
		w.cancel()
	}

	if err := w.nw.Close(); err != nil {
		w.onError(err)
	}

	for _, handler := range w.handlers {
		if err := handler.Stop(); err != nil {
			w.onError(err)
		}
	}
}

func (w *Watcher) selectEvent(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-w.nw.Events:
			w.onEvent(e)
		case err := <-w.nw.Errors:
			w.onError(err)
		}
	}
}

func (w *Watcher) add(dir string, file os.FileInfo) error {
	path := filepath.Join(dir, file.Name())

	if err := w.nw.Add(path); err != nil {
		return err
	}

	if file.IsDir() {
		fis, err := ioutil.ReadDir(path)
		if err != nil {
			return err
		}
		for _, fi := range fis {
			if err := w.add(path, fi); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *Watcher) onEvent(ev fsnotify.Event) {
	name, err := filepath.Rel(w.path, ev.Name)
	if err != nil {
		name = ev.Name
	}
	e := Event{Name: name, Op: Op(ev.Op)}

	if e.Op.Is(Create) {
		if stat, _ := os.Stat(e.Name); stat != nil && stat.IsDir() {
			log.Infof("增加文件夹监控: %s", e.Name)
			if err := w.add(filepath.Dir(e.Name), stat); err != nil {
				w.onError(fmt.Errorf("增加文件夹监控出错: %w", err))
			}
		}
	}

	if e.Op.Is(Create, Write, Remove, Rename) {
		for _, handler := range w.handlers {
			if handler.Match(e) {
				log.Infof("%s -> %s", e.Op, e.Name)
				go handler.Execute()
			}
		}
	}
}
