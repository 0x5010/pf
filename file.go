package pf

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

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

var defaultPieceCount int

func init() {
	defaultPieceCount = runtime.NumCPU()
}

// PF progress file
type PF struct {
	mutex      sync.RWMutex
	file       *os.File
	Filepath   string
	Hash       string
	FileSize   int
	PieceSize  int
	PieceCount int
	Progress   *Progress
	checked    bool
	finish     chan struct{}
}

// PFOption initialization option
type PFOption func(*PF)

// SetPieceSize set PF PieceSize
func SetPieceSize(pieceSize int) PFOption {
	return func(pf *PF) {
		pf.PieceCount = (pf.FileSize + pieceSize - 1) / pieceSize
		pf.PieceSize = pieceSize
	}
}

// SetPieceCount set PF PieceCount
func SetPieceCount(pieceCount int) PFOption {
	return func(pf *PF) {
		pf.PieceCount = pieceCount
		pf.PieceSize = (pf.FileSize + pieceCount - 1) / pieceCount
	}
}

// SetHash set PF Hash
func SetHash(hash string) PFOption {
	return func(pf *PF) {
		pf.Hash = hash
	}
}

// New initialization PF
func New(filename string, fileSize int, opts ...PFOption) (*PF, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	piecesize := (fileSize + defaultPieceCount - 1) / defaultPieceCount
	pf := &PF{
		file:       file,
		Filepath:   filename,
		FileSize:   fileSize,
		PieceSize:  piecesize,
		PieceCount: defaultPieceCount,
		finish:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(pf)
	}

	pf.Progress = NewProgress(pf.PieceCount)
	go pf.run()
	return pf, nil
}

// Write write piece data to file
func (pf *PF) Write(index int, data []byte) error {
	pf.mutex.Lock()
	defer pf.mutex.Unlock()

	if pf.checked {
		return nil
	}

	if ok, err := pf.Progress.Contains(index); ok || err != nil {
		return err
	}
	_, err := pf.file.WriteAt(data, int64(index)*int64(pf.PieceSize))
	if err != nil {
		fmt.Println("err write at")
		return err
	}
	pf.Progress.Add(index)
	return nil
}

func (pf *PF) run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for !pf.Checked() {
		select {
		case <-ticker.C:
			if pf.Progress.Check() {
				err := pf.fileCheck()
				pf.checked = true
				pf.file.Close()
				if err == nil {
					fmt.Println("ok")
				} else {
					fmt.Println("no ok: ", err.Error())
				}
				close(pf.finish)
				return
			}
		}
	}
}

// WaitFinish wait all done
func (pf *PF) WaitFinish() {
	pf.mutex.RLock()
	defer pf.mutex.RUnlock()
	if pf.checked {
		return
	}
	<-pf.finish
}

func (pf *PF) fileCheck() error {
	pf.mutex.Lock()
	defer pf.mutex.Unlock()

	if pf.checked {
		return nil
	}

	fi, err := os.Stat(pf.Filepath)
	if err != nil {
		return err
	}

	filesize := int64(pf.FileSize)
	if fi.Size() != filesize {
		err = pf.file.Truncate(filesize)
		if err != nil {
			return err
		}
	}

	if pf.hashCheck() {
		return nil
	}
	return fmt.Errorf("hash no match")
}

func (pf *PF) hashCheck() bool {
	if len(pf.Hash) == 0 {
		return true
	}
	data, err := ioutil.ReadFile(pf.Filepath)
	if err != nil {
		log.Println(err)
		return false
	}
	cur := fmt.Sprintf("%x", md5.Sum(data))
	if cur == pf.Hash {
		log.Println("check success:", pf.Hash)
		return true
	}

	log.Println("check failed:", cur, pf.Hash)
	return false
}

// Checked check all done
func (pf *PF) Checked() bool {
	pf.mutex.RLock()
	defer pf.mutex.RUnlock()
	return pf.checked
}
