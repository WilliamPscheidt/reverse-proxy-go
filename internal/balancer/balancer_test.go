package balancer

import (
	"fmt"
	"net/url"
	"sync"
	"testing"

	"github.com/WilliamPscheidt/reverse-proxy-go/internal/proxy"
)

func makeTestPool(n int) *proxy.BackendPool {
	pool := proxy.NewBackendPool()
	for i := 0; i < n; i++ {
		u, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:808%d", i+1))
		b := proxy.NewBackend(u, fmt.Sprintf("127.0.0.1:808%d", i+1), 1)
		pool.AddBackend(b)
	}
	return pool
}

func TestRoundRobin(t *testing.T) {
	pool := makeTestPool(3)
	rr := NewRoundRobin()

	var selected []string
	for i := 0; i < 6; i++ {
		b, err := rr.NextHTTP(pool, nil, "127.0.0.1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		selected = append(selected, b.Addr)
	}

	expected := []string{
		"127.0.0.1:8081",
		"127.0.0.1:8082",
		"127.0.0.1:8083",
		"127.0.0.1:8081",
		"127.0.0.1:8082",
		"127.0.0.1:8083",
	}

	for i, exp := range expected {
		if selected[i] != exp {
			t.Errorf("step %d: expected %s, got %s", i, exp, selected[i])
		}
	}
}

func TestLeastConnections(t *testing.T) {
	pool := makeTestPool(3)
	lc := NewLeastConnections()

	backends := pool.GetBackends()
	backends[0].IncConns()
	backends[0].IncConns() // 2 conns
	backends[1].IncConns() // 1 conn
	// backends[2] has 0 conns

	b, err := lc.NextHTTP(pool, nil, "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Addr != "127.0.0.1:8083" {
		t.Errorf("expected 127.0.0.1:8083, got %s", b.Addr)
	}

	// Now increment backend 2 so it has 2 conns
	backends[2].IncConns()
	backends[2].IncConns()

	// Backend 1 has 1 conn, which is now the least
	b2, err := lc.NextHTTP(pool, nil, "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b2.Addr != "127.0.0.1:8082" {
		t.Errorf("expected 127.0.0.1:8082, got %s", b2.Addr)
	}
}

func TestIPHash(t *testing.T) {
	pool := makeTestPool(3)
	ih := NewIPHash()

	clientA := "192.168.1.100:4521"
	b1, err := ih.NextHTTP(pool, nil, clientA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Repeated requests with same IP must hit the exact same backend
	for i := 0; i < 20; i++ {
		b2, _ := ih.NextHTTP(pool, nil, clientA)
		if b2.Addr != b1.Addr {
			t.Fatalf("inconsistent routing for same IP: got %s, want %s", b2.Addr, b1.Addr)
		}
	}
}

func TestNoHealthyBackends(t *testing.T) {
	pool := makeTestPool(2)
	for _, b := range pool.GetBackends() {
		b.SetAlive(false)
	}

	rr := NewRoundRobin()
	_, err := rr.NextHTTP(pool, nil, "127.0.0.1")
	if err != ErrNoHealthyBackends {
		t.Errorf("expected ErrNoHealthyBackends, got %v", err)
	}
}

func BenchmarkRoundRobin(b *testing.B) {
	pool := makeTestPool(5)
	rr := NewRoundRobin()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = rr.NextHTTP(pool, nil, "127.0.0.1")
		}
	})
}

func BenchmarkLeastConnections(b *testing.B) {
	pool := makeTestPool(5)
	lc := NewLeastConnections()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = lc.NextHTTP(pool, nil, "127.0.0.1")
		}
	})
}

func BenchmarkIPHash(b *testing.B) {
	pool := makeTestPool(5)
	ih := NewIPHash()

	ips := []string{"10.0.0.1", "10.0.0.2", "192.168.1.50", "172.16.0.4"}
	var wg sync.WaitGroup
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wg
		_, _ = ih.NextHTTP(pool, nil, ips[i%len(ips)])
	}
}
