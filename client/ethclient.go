/*
 * Copyright (C) 2020 The "MysteriumNetwork/payments" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package client

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/ethclient"
)

// NewReconnectableEthClient creates new ethereum client that can reconnect.
func NewReconnectableEthClient(address string) (*ReconnectableEthClient, error) {
	ec, err := ethclient.Dial(address)
	if err != nil {
		return nil, fmt.Errorf("ethereum client failed to connect: %w", err)
	}

	return &ReconnectableEthClient{
		address: address,
		client:  ec,
	}, nil
}

// ReconnectableEthClient is a ethereum client that can reconnect.
type ReconnectableEthClient struct {
	address string
	mu      sync.Mutex
	client  *ethclient.Client
}

// Client returns the currently connected ethereum client.
func (c *ReconnectableEthClient) Client() *ethclient.Client {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.client
}

// Reconnect creates new ethereum client and replaces the current one.
func (c *ReconnectableEthClient) Reconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	client, err := ethclient.Dial(c.address)
	if err != nil {
		return fmt.Errorf("ethereum client failed to dial: %w", err)
	}

	c.client.Close()
	c.client = client

	return nil
}
