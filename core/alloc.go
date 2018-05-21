package core

import (
	"sync"
	"time"
	"errors"
)

// 号段：[left,right)
type Segment struct {
	offset int64	// 消费偏移
	left int64	// 左区间
	right int64	// 右区间
}

// 关联到bizTag的号码池
type BizAlloc struct {
	mutex sync.Mutex
	bizTag string // 业务标识
	segments []*Segment // 双Buffer, 最少0个, 最多2个号段在内存
	isAllocating bool // 是否正在分配中(远程获取)
	waiting []chan byte // 因号码池空而挂起等待的客户端
}

// 全局分配器, 管理所有的biz号码分配
type Alloc struct {
	mutex sync.Mutex
	bizMap map[string]*BizAlloc
}

var GAlloc *Alloc

func InitAlloc() (err error) {
	GAlloc = &Alloc{
		bizMap: map[string]*BizAlloc{},
	}
	return
}

func (bizAlloc *BizAlloc) leftCount() (count int64) {
	for i := 0; i < len(bizAlloc.segments); i++ {
		count += bizAlloc.segments[i].right - bizAlloc.segments[i].left - bizAlloc.segments[i].offset
	}
	return count
}

func (bizAlloc *BizAlloc) leftCountWithMutex() (count int64) {
	bizAlloc.mutex.Lock()
	defer bizAlloc.mutex.Unlock()
	return bizAlloc.leftCount()
}

func (bizAlloc *BizAlloc) newSegment() (seg *Segment, err error) {
	var (
		maxId int64
		step int64
	)
	if maxId, step, err = GMysql.NextId(bizAlloc.bizTag); err != nil {
		return
	}
	seg = &Segment{}
	seg.left = maxId - step
	seg.right = maxId
	return
}

func (bizAlloc *BizAlloc) wakeup() {
	var (
		waitChan chan byte
	)
	for _,  waitChan = range bizAlloc.waiting {
		close(waitChan)
	}
	bizAlloc.waiting = bizAlloc.waiting[:0]
}

// 分配号码段, 直到足够2个segment, 否则始终不会退出
func (bizAlloc *BizAlloc) fillSegments() {
	var (
		failTimes int64 // 连续分配失败次数
		seg *Segment
		err error
	)
	for {
		bizAlloc.mutex.Lock()
		if len(bizAlloc.segments) <= 1 {	// 只剩余<=1段, 那么继续获取新号段
			bizAlloc.mutex.Unlock()
			// 请求mysql获取号段
			if seg, err = bizAlloc.newSegment(); err != nil {
				failTimes++
				if failTimes > 3 {	// 连续失败超过3次则停止分配
					bizAlloc.mutex.Lock()
					bizAlloc.wakeup() // 唤醒等待者, 让它们立马失败
					goto LEAVE
				}
			} else {
				failTimes = 0 // 分配成功则失败次数重置为0
				// 新号段补充进去
				bizAlloc.mutex.Lock()
				bizAlloc.segments = append(bizAlloc.segments, seg)
				bizAlloc.wakeup() // 尝试唤醒等待资源的调用
				if len(bizAlloc.segments) > 1 { // 已生成2个号段, 停止继续分配
					goto LEAVE
				} else {
					bizAlloc.mutex.Unlock()
				}
			}
		} else {
			// never reach
			break
		}
	}

LEAVE:
	bizAlloc.isAllocating = false
	bizAlloc.mutex.Unlock()
}

func (bizAlloc *BizAlloc) popNextId() (nextId int64) {
	nextId = bizAlloc.segments[0].left + bizAlloc.segments[0].offset
	bizAlloc.segments[0].offset++
	if nextId + 1 >= bizAlloc.segments[0].right {
		bizAlloc.segments = append(bizAlloc.segments[:0], bizAlloc.segments[1:]...) // 弹出第一个seg, 后续seg向前移动
	}
	return
}

func (bizAlloc *BizAlloc) nextId() (nextId int64, err error) {
	var (
		waitChan chan byte
		waitTimer *time.Timer
		hasId = false
	)

	bizAlloc.mutex.Lock()
	defer bizAlloc.mutex.Unlock()

	// 1, 有剩余号码, 立即分配返回
	if bizAlloc.leftCount() != 0 {
		nextId = bizAlloc.popNextId()
		hasId = true
	}

	// 2, 段<=1个, 启动补偿线程
	if len(bizAlloc.segments) <= 1 && !bizAlloc.isAllocating {
		bizAlloc.isAllocating = true
		go bizAlloc.fillSegments()
	}

	// 分配到号码, 立即退出
	if hasId {
		return
	}

	// 3, 没有剩余号码, 此时补偿线程一定正在运行, 等待其至多一段时间
	waitChan = make(chan byte, 1)
	bizAlloc.waiting = append(bizAlloc.waiting, waitChan)	// 排队等待唤醒

	// 释放锁, 等待补偿线程唤醒
	bizAlloc.mutex.Unlock()

	waitTimer = time.NewTimer(2 * time.Second) // 最多等待2秒
	select {
	case <- waitChan:
	case <- waitTimer.C:
	}

	// 4, 再次上锁尝试获取号码
	bizAlloc.mutex.Lock()
	if bizAlloc.leftCount() != 0 {
		nextId = bizAlloc.popNextId()
	} else {
		err = errors.New("no available id")
	}
	return
}

func (alloc *Alloc) NextId(bizTag string) (nextId int64, err error) {
	var (
		bizAlloc *BizAlloc
		exist bool
	)

	alloc.mutex.Lock()
	if bizAlloc, exist = alloc.bizMap[bizTag]; !exist {
		bizAlloc = &BizAlloc{
			bizTag: bizTag,
			segments: make([]*Segment, 0),
			isAllocating: false,
			waiting: make([]chan byte, 0),
		}
		alloc.bizMap[bizTag] = bizAlloc
	}
	alloc.mutex.Unlock()

	nextId, err = bizAlloc.nextId()
	return
}

func (alloc *Alloc) LeftCount(bizTag string) (leftCount int64) {
	var (
		bizAlloc *BizAlloc
	)

	alloc.mutex.Lock()
	bizAlloc, _ = alloc.bizMap[bizTag]
	alloc.mutex.Unlock()

	if bizAlloc != nil {
		leftCount = bizAlloc.leftCountWithMutex()
	}
	return
}
