package tbutil

import (
	"context"
	"fmt"

	tb "github.com/tigerbeetle/tigerbeetle-go"
	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

// ClientPool manages a fixed set of TigerBeetle clients.
type ClientPool struct {
	clients   []tb.Client
	available chan tb.Client
}

// NewClientPool creates a pool with the requested number of sessions.
func NewClientPool(clusterID uint32, addresses []string, sessions int) (*ClientPool, error) {
	if sessions <= 0 {
		sessions = 1
	}
	clients := make([]tb.Client, 0, sessions)
	available := make(chan tb.Client, sessions)
	cluster := tbtypes.ToUint128(uint64(clusterID))
	for i := 0; i < sessions; i++ {
		client, err := tb.NewClient(cluster, addresses)
		if err != nil {
			for _, c := range clients {
				c.Close()
			}
			return nil, fmt.Errorf("create TB client: %w", err)
		}
		clients = append(clients, client)
		available <- client
	}
	return &ClientPool{clients: clients, available: available}, nil
}

// Acquire returns a client from the pool or an error on context cancellation.
func (p *ClientPool) Acquire(ctx context.Context) (tb.Client, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case client := <-p.available:
		return client, nil
	}
}

// Release returns a client to the pool.
func (p *ClientPool) Release(client tb.Client) {
	if client == nil {
		return
	}
	p.available <- client
}

// Close shuts down all clients in the pool.
func (p *ClientPool) Close() error {
	for _, client := range p.clients {
		client.Close()
	}
	return nil
}
