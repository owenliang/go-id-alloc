package core

import (
	"sync"
	"time"
	"errors"
)

// [left,right)
type Segment struct {
	offset int64
	left int64 // (ID - 1) * SegmentSize
	right int64	// left + SegmentSize
}

type Alloc struct {
	mutex sync.Mutex
	segments []*Segment // ring buffer better
}

var GAlloc *Alloc

func InitAlloc() error {
	GAlloc = &Alloc{
		segments: make([]*Segment, 0, 2),
	}
	for i := 0; i < 2; i++ {
		if seg, err := GAlloc.newSegment(); err != nil {
			return err
		} else {
			GAlloc.segments = append(GAlloc.segments, seg)
		}
	}
	go GAlloc.fillSegment()
	return nil
}

func (alloc *Alloc)newSegment() (*Segment, error) {
	id, err := GMysql.NextId()
	if err != nil {
		return nil, err
	}
	seg := Segment{}
	seg.left = (id - 1) * int64(GConf.SegmentSize)
	seg.right = seg.left + int64(GConf.SegmentSize)
	return &seg, nil
}

func (alloc *Alloc)fillSegment() {
	for {
		time.Sleep(time.Duration(1) * time.Millisecond)

		alloc.mutex.Lock()
		if len(alloc.segments) <= 1 {
			alloc.mutex.Unlock()
			if seg, err := alloc.newSegment(); err != nil {
				continue
			} else {
				alloc.mutex.Lock()
				alloc.segments = append(alloc.segments, seg)
				alloc.mutex.Unlock()
			}
		} else {
			alloc.mutex.Unlock()
		}
	}
}

func (alloc *Alloc)NextId() (int64, error) {
	alloc.mutex.Lock()
	defer alloc.mutex.Unlock()

	if len(alloc.segments) > 0 {
		id := alloc.segments[0].left + alloc.segments[0].offset
		alloc.segments[0].offset++
		if id + 1 >= alloc.segments[0].right {
			alloc.segments = append(alloc.segments[:0], alloc.segments[1:]...)
		}
		return id, nil
	} else {
		return 0, errors.New("no more id")
	}
}

func (alloc *Alloc)LeftCount() (int64) {
	alloc.mutex.Lock()
	defer alloc.mutex.Unlock()

	var count int64 = 0

	for i := 0; i < len(alloc.segments); i++ {
		count += alloc.segments[i].right - alloc.segments[i].left - alloc.segments[i].offset
	}
	return count
}

