/*
 *
 * Created by 0x5010 on 2018/08/27.
 * pf
 * https://github.com/0x5010/pf
 *
 * Copyright 2018 0x5010.
 * Licensed under the MIT license.
 *
 */
package pf

import (
	"errors"
	"io"
	"sync"

	"github.com/RoaringBitmap/roaring"
)

var (
	// ErrOutOfBounds bitmap out of bounds
	ErrOutOfBounds = errors.New("out of bounds")
)

// Progress progress with bitmap
type Progress struct {
	mutex  sync.RWMutex
	size   uint32
	good   uint32
	bitset *roaring.Bitmap
	runing *roaring.Bitmap
}

// NewProgress new Progress
func NewProgress(n int) *Progress {
	rb := roaring.New()
	rb.AddInt(n)
	return &Progress{
		size:   uint32(n),
		bitset: rb,
		runing: rb.Clone(),
	}
}

// LoadProgress load Progress from io.Reader
func LoadProgress(buf io.Reader) *Progress {
	rb := roaring.New()
	rb.ReadFrom(buf)
	return &Progress{
		size:   rb.Maximum(),
		good:   uint32(rb.GetCardinality()),
		bitset: rb,
		runing: rb.Clone(),
	}
}

// Add finish x piece to progress
func (p *Progress) Add(x int) {
	if x < 0 || uint32(x) >= p.size {
		return
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.bitset.AddInt(x)
}

// Contains check x finished
func (p *Progress) Contains(x int) (bool, error) {
	if x < 0 || uint32(x) >= p.size {
		return false, ErrOutOfBounds
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.bitset.ContainsInt(x), nil
}

// Remove unfinish x piece to progress
func (p *Progress) Remove(x int) {
	if x < 0 || uint32(x) >= p.size {
		return
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.bitset.Remove(uint32(x))
}

// Check check progress finish
func (p *Progress) Check() bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.good = uint32(p.bitset.GetCardinality()) - 1
	return p.finish()
}

func (p *Progress) finish() bool {
	return p.size == p.good
}

// FindFirstClear find first unfinshed piece in progress, mask x runing
func (p *Progress) FindFirstClear() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	x := 0
	i := p.runing.Iterator()
	for i.HasNext() {
		if i.Next() > uint32(x) {
			p.runing.AddInt(x)
			return x
		}
		x++
	}
	return -1
}

// Clear if runing piece fail, clear the runing piece in progress
func (p *Progress) Clear(x int) {
	if x < 0 || uint32(x) >= p.size {
		return
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.runing.Remove(uint32(x))
}
