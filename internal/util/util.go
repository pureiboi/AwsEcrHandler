package util

import (
	"log"
	"sync"
)

func CheckError(err error) {
	if err != nil {
		log.Panic(err)
	}
}

type ImageJobCounting struct {
	Mu      *sync.Mutex
	Counter int
}

func (c *ImageJobCounting) Inc(num int) {
	c.Mu.Lock()
	c.Counter += num
	c.Mu.Unlock()
}
