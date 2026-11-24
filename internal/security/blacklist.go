package security

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Entry represents a blacklisted IP address with metadata.
type Entry struct {
	IP        string    `json:"ip"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// Blacklist manages blocked IP addresses with high performance in-memory lookups
// and optional distributed persistence via Redis.
type Blacklist struct {
	mu      sync.RWMutex
	blocked map[string]Entry
	rdb     *redis.Client
}

// NewBlacklist creates a new Blacklist manager.
func NewBlacklist(rdb *redis.Client) *Blacklist {
	bl := &Blacklist{
		blocked: make(map[string]Entry),
		rdb:     rdb,
	}

	// Load existing entries from Redis if connected
	if rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		keys, err := rdb.Keys(ctx, "proxy:blacklist:*").Result()
		if err == nil {
			for _, k := range keys {
				ip := strings.TrimPrefix(k, "proxy:blacklist:")
				reason, _ := rdb.Get(ctx, k).Result()
				bl.blocked[ip] = Entry{
					IP:        ip,
					Reason:    reason,
					CreatedAt: time.Now(),
				}
			}
		}
	}

	return bl
}

// SanitizeIP extracts the IP address from a host:port string or bare IP.
func SanitizeIP(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(addr)
	if err == nil && host != "" {
		return host
	}
	return addr
}

// IsBlocked checks if an IP is in the blacklist. Lock-free/read-lock for ultra-high concurrency.
func (b *Blacklist) IsBlocked(addr string) bool {
	ip := SanitizeIP(addr)
	if ip == "" {
		return false
	}

	b.mu.RLock()
	_, exists := b.blocked[ip]
	b.mu.RUnlock()
	return exists
}

// Block adds an IP address to the blacklist.
func (b *Blacklist) Block(addr string, reason string) Entry {
	ip := SanitizeIP(addr)
	if reason == "" {
		reason = "Bloqueado pelo administrador"
	}

	entry := Entry{
		IP:        ip,
		Reason:    reason,
		CreatedAt: time.Now(),
	}

	b.mu.Lock()
	b.blocked[ip] = entry
	b.mu.Unlock()

	// Persist to Redis if available
	if b.rdb != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = b.rdb.Set(ctx, "proxy:blacklist:"+ip, reason, 0).Err()
		}()
	}

	return entry
}

// Unblock removes an IP address from the blacklist.
func (b *Blacklist) Unblock(addr string) bool {
	ip := SanitizeIP(addr)

	b.mu.Lock()
	_, exists := b.blocked[ip]
	delete(b.blocked, ip)
	b.mu.Unlock()

	if exists && b.rdb != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = b.rdb.Del(ctx, "proxy:blacklist:"+ip).Err()
		}()
	}

	return exists
}

// List returns all currently blacklisted entries.
func (b *Blacklist) List() []Entry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	entries := make([]Entry, 0, len(b.blocked))
	for _, e := range b.blocked {
		entries = append(entries, e)
	}
	return entries
}

// Count returns the number of blacklisted IPs.
func (b *Blacklist) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.blocked)
}
