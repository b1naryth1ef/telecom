package telecom

import (
	"errors"
)

var (
	ErrDone = errors.New("Playable Done")
)

type Playable interface {
	Output() (chan []byte, error)
	Close()
}

type BasicPlayable struct {
	d bool
	c chan []byte
}

func NewBasicPlayable() *BasicPlayable {
	return &BasicPlayable{
		false,
		make(chan []byte, 0),
	}
}

func (bp *BasicPlayable) Output() (chan []byte, error) {
	if bp.d {
		return nil, ErrDone
	}

	return bp.c, nil
}

func (bp *BasicPlayable) Input() (chan []byte, error) {
	if bp.d {
		return nil, ErrDone
	}

	return bp.c, nil
}

func (bp *BasicPlayable) Close() {
	bp.d = false
	close(bp.c)
}
