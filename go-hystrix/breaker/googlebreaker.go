package breaker

import (
	"examples/go-hystrix/collection"
	"examples/go-hystrix/mathx"
	"math"
	"time"
)

const (
	// 1000 / 40 = 250ms for bucket duration
	window     = time.Second * 10
	buckets    = 40
	k          = 1.5
	protection = 5
)

type googleBreaker struct {
	// 敏感度
	k float64
	// 滑动窗口
	stat *collection.RollingWindow
	// 概率生成器：随机产生[0, 1] 之间的双精度浮点数
	proba *mathx.Proba
}

func newGoogleBreaker() *googleBreaker {
	bucketDuration := time.Duration(int64(window) / int64(buckets))
	st := collection.NewRollingWindow(buckets, bucketDuration)
	return &googleBreaker{
		stat:  st,
		k:     k,
		proba: mathx.NewProba(),
	}
}

// allow 熔断方法
// 简单场景直接判断对象是否被熔断，执行请求后必须手动上报执行结果至熔断器
// 返回一个 promise 异步回调对象，可以由开发者自行决定是否上报结果到熔断器
func (b *googleBreaker) allow() (internalPromise, error) {
	if err := b.accept(); err != nil {
		return nil, err
	}
	return googlePromise{
		b: b,
	}, nil
}

// doReq 熔断方法
// 复杂场景下，支持自定义快速失败，自定义判定请求是否成功的熔断方法，自动上报执行结果至熔断器
// req - 熔断对象方法
// fallback - 自定义快速失败函数，可对熔断产生的 err 包装后返回
// acceptable - 对本次未熔断的请求结果进行自定义判定是否成功，例如针对 http.code，rpc.code 等
func (b *googleBreaker) doReq(req func() error, fallback func(err error) error, acceptable Acceptable) error {
	if err := b.accept(); err != nil {
		if fallback != nil {
			return fallback(err)
		}

		return err
	}

	defer func() {
		if e := recover(); e != nil {
			b.markFailure()
			panic(e)
		}
	}()

	err := req()
	if acceptable(err) {
		b.markSuccess()
	} else {
		b.markFailure()
	}
	return err
}

// 根据最近一段时间的请求数据计算是否熔断
func (b *googleBreaker) accept() error {
	// 获取最近一段时间内的统计数据
	accepts, total := b.history()
	// 计算动态熔断概率
	weightedAccepts := b.k * float64(accepts)
	dropRatio := math.Max(0, (float64(total-protection)-weightedAccepts)/float64(total+1))
	// 概率为0，通过
	if dropRatio <= 0 {
		return nil
	}

	// 随机产生 [0.0, 1.0] 之间的随机数与上面的熔断概率比较
	// 若随机数比熔断概率小则进行熔断
	if b.proba.TrueOnProba(dropRatio) {
		return ErrServiceUnavailable
	}

	return nil
}

func (b *googleBreaker) history() (accepts, total int64) {
	b.stat.Reduce(func(b *collection.Bucket) {
		accepts += int64(b.Sum)
		total += b.Count
	})
	return
}

func (b *googleBreaker) markSuccess() {
	b.stat.Add(1)
}

func (b *googleBreaker) markFailure() {
	b.stat.Add(0)
}

type googlePromise struct {
	b *googleBreaker
}

func (p googlePromise) Accept() {
	p.b.markSuccess()
}

func (p googlePromise) Reject() {
	p.b.markFailure()
}
