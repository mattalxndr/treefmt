package caching

import (
	"fmt"
)

type noOpTx struct {
}

func (n noOpTx) Commit() error {
	return nil
}

func (n noOpTx) Rollback() error {
	return nil
}

type NoOpCache struct {
}

func (n NoOpCache) BeginRead() (Tx, error) {
	return noOpTx{}, nil
}

func (n NoOpCache) View(f func(tx Tx) error) error {
	return f(noOpTx{})
}

func (n NoOpCache) Update(f func(tx Tx) error) error {
	return f(noOpTx{})
}

func (n NoOpCache) tx(tx Tx) (*noOpTx, error) {
	switch v := tx.(type) {
	case noOpTx:
		return &v, nil
	default:
		return nil, fmt.Errorf("tx must be of type NoOpTx")
	}
}

func (n NoOpCache) GetPath(tx Tx, _ string) (*Entry, error) {
	// enforce the correct tx type
	_, err := n.tx(tx)
	return nil, err
}

func (n NoOpCache) UpdatePath(tx Tx, _ string, _ *Entry) error {
	// enforce the correct tx type
	_, err := n.tx(tx)
	return err
}

func (n NoOpCache) RemoveAllPaths(tx Tx) error {
	// enforce the correct tx type
	_, err := n.tx(tx)
	return err
}

//func (n NoOpCache) HaveFormattersChanged(tx Tx, _ map[string]*format.Formatter) (bool, error) {
//	// enforce the correct tx type
//	_, err := n.tx(tx)
//	return true, err
//}

func (n NoOpCache) Close() error {
	return nil
}
