// session_cache.go 提供 TLS session resumption 的客户端会话缓存。
//
// 真实客户端（Node.js / rustls）都会复用会话票据，每次都做完整握手反而是
// 自动化客户端的显著特征。uTLS 的 sessionController 在 ClientSessionCache
// 非空时会自动注入/消费 SessionTicket 与 pre_shared_key 扩展完成恢复。
package tlsfingerprint

import (
	"container/list"
	"sync"

	utls "github.com/refraction-networking/utls"
)

// sessionCacheCapacity 全局 session 缓存容量上限（条目数），超出后淘汰最久未使用条目。
// 单条目约数百字节，1024 条对内存影响可忽略，同时足以覆盖全部上游主机×模板组合。
const sessionCacheCapacity = 1024

// sharedSessionCache 进程级共享的 LRU session 缓存（并发安全）。
var sharedSessionCache = newLRUSessionCache(sessionCacheCapacity)

// clientSessionCacheForProfile 返回按模板名隔离的缓存视图。
// 隔离目的：避免 Node.js 指纹的连接复用 rustls 模板协商出的 session（跨层矛盾），
// 反之亦然。profile 为 nil 或无名时使用 "default" 前缀。
func clientSessionCacheForProfile(profile *Profile) utls.ClientSessionCache {
	name := "default"
	if profile != nil && profile.Name != "" {
		name = profile.Name
	}
	return &profileSessionCache{prefix: name + "|", store: sharedSessionCache}
}

// profileSessionCache 给共享缓存的键加模板名前缀，实现模板间隔离。
// uTLS 传入的 sessionKey 只含对端地址，不含模板维度，因此在这里补充。
type profileSessionCache struct {
	prefix string
	store  *lruSessionCache
}

func (c *profileSessionCache) Get(sessionKey string) (*utls.ClientSessionState, bool) {
	return c.store.Get(c.prefix + sessionKey)
}

func (c *profileSessionCache) Put(sessionKey string, cs *utls.ClientSessionState) {
	c.store.Put(c.prefix+sessionKey, cs)
}

// lruSessionCache 容量有界、并发安全的 LRU 实现。
// uTLS 自带的 NewLRUClientSessionCache 不支持键前缀隔离，因此自行实现。
type lruSessionCache struct {
	mu      sync.Mutex
	cap     int
	items   map[string]*list.Element
	lruList *list.List // front = 最近使用
}

type lruSessionEntry struct {
	key   string
	state *utls.ClientSessionState
}

func newLRUSessionCache(capacity int) *lruSessionCache {
	return &lruSessionCache{
		cap:     capacity,
		items:   make(map[string]*list.Element),
		lruList: list.New(),
	}
}

func (c *lruSessionCache) Get(key string) (*utls.ClientSessionState, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	// comma-ok：list 元素只由本包写入，类型恒为 *lruSessionEntry；断言失败视为未命中
	entry, ok := el.Value.(*lruSessionEntry)
	if !ok {
		return nil, false
	}
	c.lruList.MoveToFront(el)
	return entry.state, true
}

// Put 写入会话；cs 为 nil 时按接口约定删除该条目。
func (c *lruSessionCache) Put(key string, cs *utls.ClientSessionState) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cs == nil {
		if el, ok := c.items[key]; ok {
			c.lruList.Remove(el)
			delete(c.items, key)
		}
		return
	}

	if el, ok := c.items[key]; ok {
		if entry, ok := el.Value.(*lruSessionEntry); ok {
			entry.state = cs
			c.lruList.MoveToFront(el)
		}
		return
	}

	c.items[key] = c.lruList.PushFront(&lruSessionEntry{key: key, state: cs})
	if c.lruList.Len() > c.cap {
		// 淘汰最久未使用的条目
		if back := c.lruList.Back(); back != nil {
			if entry, ok := back.Value.(*lruSessionEntry); ok {
				delete(c.items, entry.key)
			}
			c.lruList.Remove(back)
		}
	}
}
