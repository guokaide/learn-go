package collection

import (
	"examples/go-hystrix/timex"
	"sync"
	"time"
)

// RollingWindowOption 用于自定义 RollingWindow
type RollingWindowOption func(rollingWindow *RollingWindow)

type RollingWindow struct {
	// 互斥锁
	lock sync.Mutex
	// 滑动窗口存储
	win *window
	// 滑动窗口大小
	size int
	// 滑动窗口单元时间间隔
	interval time.Duration
	// 滑动窗口应该写入 bucket 的偏移量
	offset int
	// 汇总数据时，是否忽略当前正在写入桶的数据
	ignoreCurrent bool
	// 最后写入桶的时间(最后一个当前桶的开始时间)
	lastTime time.Duration
}

func NewRollingWindow(size int, interval time.Duration, opts ...RollingWindowOption) *RollingWindow {
	if size < 1 {
		panic("size must be greater than 0")
	}

	w := &RollingWindow{
		size:     size,
		interval: interval,
		win:      newWindow(size),
		lastTime: timex.Now(),
	}

	for _, opt := range opts {
		opt(w)
	}
	return w
}

func (rw *RollingWindow) Add(v float64) {
	rw.lock.Lock()
	defer rw.lock.Unlock()
	// 更新当前时间写入 bucket 的偏移量
	rw.updateOffset()
	rw.win.add(rw.offset, v)
}

func (rw *RollingWindow) Reduce(fn func(b *Bucket)) {
	rw.lock.Lock()
	defer rw.lock.Unlock()

	var diff int
	span := rw.span()
	if span == 0 && rw.ignoreCurrent {
		diff = rw.size - 1
	} else {
		diff = rw.size - span
	}
	if diff > 0 {
		// [rw.offset, rw.offset + span] 过期，不作统计
		offset := (rw.offset + span + 1) % rw.size
		rw.win.reduce(offset, diff, fn)
	}
}

func (rw *RollingWindow) updateOffset() {
	span := rw.span()
	if span <= 0 {
		return
	}

	offset := rw.offset
	// reset 过期的桶
	for i := 0; i < span; i++ {
		rw.win.resetBucket((offset + i + 1) % rw.size)
	}
	rw.offset = (offset + span) % rw.size
	now := timex.Now()
	// current lastTime = now - (now - lastTime) % interval
	rw.lastTime = now - (now-rw.lastTime)%rw.interval
}

// 获取过期的桶数：上一次更新的时间到当前时间经过的桶
func (rw *RollingWindow) span() int {
	offset := int(timex.Since(rw.lastTime) / rw.interval)
	if offset >= 0 && offset < rw.size {
		return offset
	}
	return rw.size
}

// 时间窗口
type window struct {
	// 每个桶代表一个时间间隔
	buckets []*Bucket
	// 窗口大小
	size int
}

func newWindow(size int) *window {
	buckets := make([]*Bucket, size)
	for i := 0; i < size; i++ {
		buckets[i] = new(Bucket)
	}
	return &window{
		buckets: buckets,
		size:    size,
	}
}

func (w *window) add(offset int, v float64) {
	w.buckets[offset%w.size].add(v)
}

func (w *window) reduce(start int, count int, fn func(b *Bucket)) {
	for i := 0; i < count; i++ {
		fn(w.buckets[(start+i)%w.size])
	}
}

func (w *window) resetBucket(offset int) {
	w.buckets[offset%w.size].reset()
}

// Bucket 存储一段时间范围的统计值
type Bucket struct {
	// 当前时间范围内值的和
	Sum float64
	// 当前桶内 Add 的次数
	Count int64
}

func (b *Bucket) add(v float64) {
	b.Sum += v
	b.Count++
}

func (b *Bucket) reset() {
	b.Sum = 0
	b.Count = 0
}
