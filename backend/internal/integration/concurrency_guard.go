package integration

import (
	"errors"
	"sync"
)

// ErrConcurrencyLimitReached 表示本进程内的运行时并发配额已满。
// 多实例唯一领取仍由数据库租约保证，Guard 仅用于本进程资源保护。
var ErrConcurrencyLimitReached = errors.New("integration concurrency limit reached")

// ConcurrencyGuard 为运行时提供平台、外部系统和接口三级并发保护。
type ConcurrencyGuard interface {
	Acquire(externalSystemID int, interfaceDefinitionID int) (release func(), err error)
}

// InMemoryConcurrencyGuard 是受控的进程内并发保护实现。
// 它不能替代 PostgreSQL 租约，只限制单个 Worker 进程的资源占用。
type InMemoryConcurrencyGuard struct {
	mu                sync.Mutex
	totalLimit        int
	perSystemLimit    int
	perInterfaceLimit int
	totalInFlight     int
	systemInFlight    map[int]int
	interfaceInFlight map[int]int
}

// NewInMemoryConcurrencyGuard 使用明确的有限配额构造 Guard，拒绝零或负数的无限并发配置。
func NewInMemoryConcurrencyGuard(totalLimit, perSystemLimit, perInterfaceLimit int) (*InMemoryConcurrencyGuard, error) {
	if totalLimit <= 0 || perSystemLimit <= 0 || perInterfaceLimit <= 0 {
		return nil, ErrConcurrencyLimitReached
	}
	return &InMemoryConcurrencyGuard{
		totalLimit: totalLimit, perSystemLimit: perSystemLimit, perInterfaceLimit: perInterfaceLimit,
		systemInFlight: make(map[int]int), interfaceInFlight: make(map[int]int),
	}, nil
}

func (g *InMemoryConcurrencyGuard) Acquire(externalSystemID int, interfaceDefinitionID int) (func(), error) {
	if g == nil || externalSystemID <= 0 || interfaceDefinitionID <= 0 {
		return nil, ErrConcurrencyLimitReached
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.totalInFlight >= g.totalLimit || g.systemInFlight[externalSystemID] >= g.perSystemLimit ||
		g.interfaceInFlight[interfaceDefinitionID] >= g.perInterfaceLimit {
		return nil, ErrConcurrencyLimitReached
	}
	g.totalInFlight++
	g.systemInFlight[externalSystemID]++
	g.interfaceInFlight[interfaceDefinitionID]++
	var once sync.Once
	return func() {
		once.Do(func() {
			g.mu.Lock()
			defer g.mu.Unlock()
			g.totalInFlight--
			g.systemInFlight[externalSystemID]--
			g.interfaceInFlight[interfaceDefinitionID]--
			if g.systemInFlight[externalSystemID] == 0 {
				delete(g.systemInFlight, externalSystemID)
			}
			if g.interfaceInFlight[interfaceDefinitionID] == 0 {
				delete(g.interfaceInFlight, interfaceDefinitionID)
			}
		})
	}, nil
}
