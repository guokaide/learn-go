package breaker

import (
	"errors"
	"examples/go-hystrix/timex"
	"fmt"
	"github.com/tal-tech/go-zero/core/mathx"
	"github.com/tal-tech/go-zero/core/proc"
	"github.com/tal-tech/go-zero/core/stat"
	"strings"
	"sync"
)

const (
	numHistoryReasons = 5
	timeFormat        = "15:04:05"
)

// ErrServiceUnavailable is returned when the Breaker state is open.
var ErrServiceUnavailable = errors.New("circuit breaker is open")

// Acceptable is the func to check if the error can be accepted.
type Acceptable func(err error) bool

// Promise interface defines the method to customize a Breaker.
type Promise interface {
	// Accept tells the Breaker that the call is successful.
	Accept()
	// Reject tells the Breaker that the call is failed.
	Reject(reason string)
}

// Option defines the method to customize a Breaker.
type Option func(breaker *circuitBreaker)

// A Breaker represents a circuit breaker.
// 熔断器接口定义 https://talkgo.org/t/topic/3035
type Breaker interface {
	// Name returns the name of the Breaker.
	Name() string

	// Allow checks if the request is allowed.
	// If allowed, a promise will be returned, the caller needs to call promise.Accept()
	// on success, or call promise.Reject() on failure.
	// If not allow, ErrServiceUnavailable will be returned.
	Allow() (Promise, error)

	// Do runs the given request if the Breaker accepts it.
	// Do returns an error instantly if the Breaker rejects the request.
	// If a panic occurs in the request, the Breaker handles it as an error
	// and causes the same panic again.
	Do(req func() error) error

	// DoWithAcceptable runs the given request if the Breaker accepts it.
	// DoWithAcceptable returns an error instantly if the Breaker rejects the request.
	// If a panic occurs in the request, the Breaker handles it as an error
	// and causes the same panic again.
	// acceptable checks if it's a successful call, even if the err is not nil.
	DoWithAcceptable(req func() error, acceptable Acceptable) error

	// DoWithFallback runs the given request if the Breaker accepts it.
	// DoWithFallback runs the fallback if the Breaker rejects the request.
	// If a panic occurs in the request, the Breaker handles it as an error
	// and causes the same panic again.
	DoWithFallback(req func() error, fallback func(err error) error) error

	// DoWithFallbackAcceptable runs the given request if the Breaker accepts it.
	// DoWithFallbackAcceptable runs the fallback if the Breaker rejects the request.
	// If a panic occurs in the request, the Breaker handles it as an error
	// and causes the same panic again.
	// acceptable checks if it's a successful call, even if the err is not nil.
	DoWithFallbackAcceptable(req func() error, fallback func(err error) error, acceptable Acceptable) error
}

func NewBreaker(opts ...Option) Breaker {
	var b circuitBreaker
	for _, opt := range opts {
		opt(&b)
	}
	if len(b.name) == 0 {
		b.name = "random name"
	}
	b.throttle = newLoggedThrottle(b.name, newGoogleBreaker())

	return &b
}

// 熔断器
// circuitBreaker
// -> throttle              熔断器接口 (代理 circuitBreaker)
// -> loggedThrottle        熔断器实现 + 日志功能 (throttle 实现类)
// -> internalThrottle      熔断器内部核心实现 (代理 loggedThrottle)
type circuitBreaker struct {
	name string
	// throttle circuitBreaker 的静态代理, 熔断功能代理代理给 throttle 实现
	throttle
}

// 熔断器接口
type throttle interface {
	allow() (Promise, error)
	doReq(req func() error, fallback func(err error) error, acceptable Acceptable) error
}

func (cb *circuitBreaker) Allow() (Promise, error) {
	return cb.throttle.allow()
}

func (cb *circuitBreaker) Do(req func() error) error {
	return cb.throttle.doReq(req, nil, defaultAcceptable)
}

func (cb *circuitBreaker) DoWithAcceptable(req func() error, acceptable Acceptable) error {
	return cb.throttle.doReq(req, nil, acceptable)
}

func (cb *circuitBreaker) DoWithFallback(req func() error, fallback func(err error) error) error {
	return cb.throttle.doReq(req, fallback, defaultAcceptable)
}

func (cb *circuitBreaker) DoWithFallbackAcceptable(req func() error, fallback func(err error) error,
	acceptable Acceptable) error {
	return cb.throttle.doReq(req, fallback, acceptable)
}

func (cb *circuitBreaker) Name() string {
	return cb.name
}

func defaultAcceptable(err error) bool {
	return err == nil
}

// loggedThrottle 带日志功能的熔断器
type loggedThrottle struct {
	name string
	// 代理对象
	internalThrottle
	// 滑动窗口，滚动收集请求失败时的错误日志
	errWin *errorWindow
}

func newLoggedThrottle(name string, t internalThrottle) loggedThrottle {
	return loggedThrottle{
		name:             name,
		internalThrottle: t,
		errWin:           new(errorWindow),
	}
}

func (lt loggedThrottle) allow() (Promise, error) {
	promise, err := lt.internalThrottle.allow()
	return promiseWithReason{
		promise: promise,
		errWin:  lt.errWin,
	}, lt.logError(err)
}

func (lt loggedThrottle) doReq(req func() error, fallback func(err error) error, acceptable Acceptable) error {
	return lt.logError(lt.internalThrottle.doReq(req, fallback, func(err error) bool {
		accept := acceptable(err)
		if !accept {
			lt.errWin.add(err.Error())
		}
		return accept
	}))
}

func (lt loggedThrottle) logError(err error) error {
	if err == ErrServiceUnavailable {
		// if circuit open, not possible to have empty error window
		stat.Report(fmt.Sprintf(
			"proc(%s/%d), callee: %s, breaker is open and requests dropped\nlast errors:\n%s",
			proc.ProcessName(), proc.Pid(), lt.name, lt.errWin))
	}

	return err
}

// 滑动窗口
type errorWindow struct {
	reasons [numHistoryReasons]string
	index   int
	count   int
	lock    sync.Mutex
}

func (ew *errorWindow) add(reason string) {
	ew.lock.Lock()
	ew.reasons[ew.index] = fmt.Sprintf("%s %s", timex.Time().Format(timeFormat), reason)
	ew.index = (ew.index + 1) % numHistoryReasons
	ew.count = mathx.MinInt(ew.count+1, numHistoryReasons)
	ew.lock.Unlock()
}

// String 格式化错误日志
func (ew *errorWindow) String() string {
	var reasons []string

	ew.lock.Lock()
	for i := ew.index - 1; i >= ew.index-ew.count; i-- {
		reasons = append(reasons, ew.reasons[(i+numHistoryReasons)%numHistoryReasons])
	}
	ew.lock.Unlock()

	return strings.Join(reasons, "\n")
}

type internalPromise interface {
	Accept()
	Reject()
}

type promiseWithReason struct {
	promise internalPromise
	errWin  *errorWindow
}

func (p promiseWithReason) Accept() {
	p.promise.Accept()
}

func (p promiseWithReason) Reject(reason string) {
	p.errWin.add(reason)
	p.promise.Reject()
}

type internalThrottle interface {
	allow() (internalPromise, error)
	doReq(req func() error, fallback func(err error) error, acceptable Acceptable) error
}
