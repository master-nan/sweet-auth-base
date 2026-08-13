package service

import "sync"

type keyedLockPool struct {
	mu    sync.Mutex
	locks map[string]*keyedLock
}

type keyedLock struct {
	mu   sync.Mutex
	refs int
}

func newKeyedLockPool() *keyedLockPool {
	return &keyedLockPool{locks: make(map[string]*keyedLock)}
}

func (p *keyedLockPool) Lock(key string) func() {
	p.mu.Lock()
	entry := p.locks[key]
	if entry == nil {
		entry = &keyedLock{}
		p.locks[key] = entry
	}
	entry.refs++
	p.mu.Unlock()

	entry.mu.Lock()
	return func() {
		entry.mu.Unlock()
		p.mu.Lock()
		entry.refs--
		if entry.refs == 0 {
			delete(p.locks, key)
		}
		p.mu.Unlock()
	}
}
