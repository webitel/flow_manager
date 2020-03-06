package app

import "sync"

type PoolTask interface {
	Execute()
}

type Pool interface {
	Resize(n int)
	Close()
	Wait()
	Exec(task PoolTask)
}

type pool struct {
	mu    sync.Mutex
	size  int
	tasks chan PoolTask
	kill  chan struct{}
	wg    sync.WaitGroup
}

func NewPool(workers int, queueCount int) Pool {
	p := &pool{
		tasks: make(chan PoolTask, queueCount),
		kill:  make(chan struct{}),
	}
	p.Resize(workers)
	return p
}

func (p *pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			task.Execute()
		case <-p.kill:
			return
		}
	}
}

func (p *pool) Resize(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for p.size < n {
		p.size++
		p.wg.Add(1)
		go p.worker()
	}
	for p.size > n {
		p.size--
		p.kill <- struct{}{}
	}
}

func (p *pool) Close() {
	close(p.tasks)
}

func (p *pool) Wait() {
	p.wg.Wait()
}

func (p *pool) Exec(task PoolTask) {
	p.tasks <- task
}

func (p *pool) ChannelJobs() chan PoolTask {
	return p.tasks
}
