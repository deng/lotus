package storage

import (
	"context"
)

type Dealer struct{}

func NewDealer() (*Dealer, error) {
	m := &Dealer{}
	return m, nil
}

func (m *Dealer) RunDealer(ctx context.Context) error {
	return nil
}

func (m *Dealer) StopDealer(ctx context.Context) error {
	return nil
}
