# learn-go

## Network

### 问题

> 1.总结几种 socket 粘包的解包方式：fix length/delimiter based/length field based frame decoder。尝试举例其应用。
>
> 2.实现一个从 socket connection 中解码出 goim 协议的解码器。

### 解决

#### 1. 粘包问题

传输控制协议 (TCP, Transmission Control Protocol) 是一种面向连接的、可靠的、基于字节流的传输层通信协议。应用层传输到传输层（TCP 协议）的数据，不是以数据包为单位向目的主机发送，而是以字节流的形式发送，这些字节流可能会被接收方重新组装成各种数据包。接收方如果因为没有正确区分不同的字节流而导致没有正确还原原来的数据，例如将 2 段表示不同信息的字节流合并或者切分 2 段表示不同信息的字节流位置不合理，就会导致粘包现象。

粘包现象出现根本原因是无法确定消息的边界。接收端在接收到字节流之后，无法判断到底接收多少个位才算是一个完整的信息，从而无法正确组装字节流，获得正确的消息。

常见的粘包的处理方式有：

* 定长分割：发送消息的时候，标记每个消息的长度，接收端就可以根据消息长度，划分不同的消息。定长分割适用于简单的网络协议，例如 MTU；
* 特殊分隔符：发送消息的时候，在数据包增加分隔符信息，接收端通过分割符，划分不同的消息。例如 HTTP 协议按照特殊分隔符分割信息；
* 定长头部 + 不定长 Body: 例如 Dubbo 协议；
* 不定长头部 + 不定长 Body: 例如 Goim 协议。不定长头部意味着需要一个额外的字段指明头部有多长。

#### 2. Goim 协议解码器

根据 Goim 协议解析即可：

> Goim 协议结构
> PacketLen 4 bytes 包长度，在数据流传输过程中，先写入整个包的长度，方便整个包的数据读取。
> HeaderLen 2 bytes 头长度，在处理数据时，会先解析头部，可以知道具体业务操作。
> Version       2 bytes 协议版本号，主要用于上行和下行数据包按版本号进行解析。
> Operation   4 bytes 业务操作码，可以按操作码进行分发数据包到具体业务当中。
> Sequence   4 bytes 序列号，数据包的唯一标记，可以做具体业务处理，或者数据包去重。
> Body           PacketLen - HeaderLen 实际业务数据，在业务层中会进行数据解码和编码。

见：[go-examples/goim/main.go](go-examples/goim/main.go)



## Redis Benchmark

### 问题

> 1.使用 redis benchmark 工具, 测试 10 20 50 100 200 1k 5k 字节 value 大小，redis get set 性能。
>
> 2.写入一定量的 kv 数据, 根据数据大小 1w-50w 自己评估, 结合写入前后的 info memory 信息 , 分析上述不同 value 大小下，平均每个 key 的占用内存空间。

### 解决

#### 1. Redis benchmark

> redis-benchmark -t set,get -n 100000 -d <size>

> ❯ redis-benchmark -t set,get -q -n 100000 -d 10
> SET: 139664.80 requests per second
> GET: 144300.14 requests per second
>
> ❯ redis-benchmark -t set,get -q -n 100000 -d 20
> SET: 140449.44 requests per second
> GET: 142857.14 requests per second
>
> ❯ redis-benchmark -t set,get -q -n 100000 -d 50
> SET: 137362.64 requests per second
> GET: 140252.45 requests per second
>
> ❯ redis-benchmark -t set,get -q -n 100000 -d 100
> SET: 138312.59 requests per second
> GET: 138888.89 requests per second
>
> ❯ redis-benchmark -t set,get -q -n 100000 -d 200
> SET: 137362.64 requests per second
> GET: 138696.25 requests per second
>
> ❯ redis-benchmark -t set,get -q -n 100000 -d 1000
> SET: 138312.59 requests per second
> GET: 139470.02 requests per second
>
> ❯ redis-benchmark -t set,get -q -n 100000 -d 5000
> SET: 132978.73 requests per second
> GET: 133868.81 requests per second
>
> ❯ redis-benchmark -t set,get -q -n 100000 -d 10000
> SET: 127064.80 requests per second
> GET: 125786.16 requests per second
>
> ❯ redis-benchmark -t set,get -q -n 100000 -d 50000
> SET: 22925.26 requests per second
> GET: 37078.23 requests per second
>
> ❯ redis-benchmark -t set,get -q -n 100000 -d 100000
> SET: 20824.66 requests per second
> GET: 34770.52 requests per second

随着 value 的大小越来越大，SET、GET 的性能会逐渐下降；当 value 的大小在 10k 字节以内时，SET、GET 的性能相对比较好，当 value 的大小超过 10k 字节之后，SET、GET 的性能下降比较厉害。

#### 2. Redis key memory analysis

分析代码见：[go-redis/main.go](go-redis/main.go)

由 [分析结果](go-redis/reports/redis-analysis-localhost-6379-0.csv) 可以看出，随着 value 的大小越来越大，其 key 的大小也会越来越大。



## Rolling Window Counter

### 问题

> 参考 Hystrix 实现一个滑动窗口计数器。

### 解决

```go
package go_hystrix

import (
	"sync"
	"time"

	"examples/go-hystrix/timex"
)

type RollingWindow struct {
	// 互斥锁
	lock sync.Mutex
	// 时间窗口
	w *Window
	// 时间窗口桶的个数
	size int
	// 时间窗口中每个桶的时间间隔
	interval time.Duration
	// 当前要写入的桶的 offset
	offset int
	// 上次更新的时间
	lastTime time.Duration
}

func (rw *RollingWindow) Add(v float64) {
	rw.lock.Lock()
	defer rw.lock.Unlock()
	rw.updateOffset()
	rw.w.Add(rw.offset, v)
}

func (rw *RollingWindow) Reduce(fn func(rw *Bucket)) {
	rw.lock.Lock()
	defer rw.lock.Unlock()

	// 未过期的元素个数
	var valid int
	span := rw.span()
	if span == 0 {
		valid = rw.size - 1
	} else {
		valid = rw.size - span
	}
	if valid > 0 {
		// 有效元素 [rw.offset + span + 1, rw.offset + valid]
		offset := (rw.offset + span + 1) % rw.size
		rw.w.Reduce(offset, valid, fn)
	}
}

func (rw *RollingWindow) updateOffset() {
	span := rw.span()
	if span <= 0 {
		return
	}
	// 过期的桶全部 Reset
	for i := 0; i < span; i++ {
		rw.w.buckets[rw.offset+span+1].Reset()
	}
	// 更新当前 offset
	rw.offset = rw.offset + span + 1
	// 更新当前 offset 对应的桶的开始时间 lastTime = now - (now - lastTime) % interval
	now := timex.Now()
	rw.lastTime = now - (now-rw.lastTime)%rw.interval
}

// span 计算从 lastTime 到当前时间过期了多少个桶
// 过期的桶的个数范围 [0, rw.size]
func (rw *RollingWindow) span() int {
	offset := int(timex.Since(rw.lastTime) / rw.interval)
	if offset >= 0 && offset < rw.size {
		return offset
	}
	return rw.size
}

// Window 某个时间周期对应的滑动窗口，用于存储一个时间周期内的统计值
type Window struct {
	buckets []*Bucket
	size    int
}

// Add 更新时间窗口某个桶的统计值
func (w *Window) Add(offset int, v float64) {
	w.buckets[offset%w.size].Add(v)
}

// Reduce 聚合一个窗口内的统计数据
func (w *Window) Reduce(start int, count int, fn func(b *Bucket)) {
	for i := 0; i < count; i++ {
		fn(w.buckets[(start+i)%w.size])
	}
}

// Bucket 某个时间范围对应的桶，用于存储该时间范围的统计值
// 用某个桶开始的时间 startTime 代表这个桶
type Bucket struct {
	// Sum 一个时间范围内统计值的和
	Sum float64
	// Count 一个时间范围内 Add 调用次数
	Count int64
}

// Add 更新某个桶的统计值
func (b *Bucket) Add(v float64) {
	b.Sum += v
	b.Count++
}

func (b *Bucket) Reset() {
	b.Sum = 0
	b.Count = 0
}
```

```go 
package go_hystrix

import "time"

// Use the long enough past time as start time, in case timex.Now() - lastTime equals 0.
var initTime = time.Now().AddDate(-1, -1, -1)

// Now returns a relative time duration since initTime
func Now() time.Duration {
	return time.Since(initTime)
}

// Since returns a diff since given d
func Since(d time.Duration) time.Duration {
	return time.Since(initTime) - d
}
```

### 参考

* https://talkgo.org/t/topic/3035



## Project Structure

### 问题

> 按照自己的构想，写一个项目满足基本的目录结构和工程，代码需要包含对数据层、业务层、API 注册，以及 main 函数对于服务的注册和启动，信号处理，使用 Wire 构建依赖。可以使用自己熟悉的框架。

### 解决

实现代码参见：[go-web-service](go-web-service)

注：由于我第一次实现 Go 的项目，因此实现主要参考：https://github.com/flycash/geekbang-go-camp

## `errgroup`

### 问题

> 基于 errgroup 实现一个 http server 的启动和关闭 ，以及 linux signal 信号的注册和处理，要保证能够一个退出，全部注销退出。

### 解决

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

type helloHandler struct {
	ctx  context.Context
	name string
}

func (h *helloHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Write([]byte(fmt.Sprintf("Hello from %s\n", h.name)))
}

func newHelloServer(
	ctx context.Context,
	name string,
	port int,
) *http.Server {

	mux := http.NewServeMux()
	handler := &helloHandler{ctx: ctx, name: name}
	mux.Handle("/", handler)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return httpServer
}

func main() {
	// setup context and signal handling
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(quit)

	g, ctx := errgroup.WithContext(ctx)

	// start servers
	server1 := newHelloServer(ctx, "server1", 8080)
	g.Go(func() error {
		log.Println("server 1 listening on port 8080")
		if err := server1.ListenAndServe();
			err != http.ErrServerClosed {
			return err
		}

		return nil
	})

	server2 := newHelloServer(ctx, "server2", 8081)
	g.Go(func() error {
		log.Println("server 2 listening on port 8081")
		if err := server2.ListenAndServe();
			err != http.ErrServerClosed {
			return err
		}

		return nil
	})

	// handle termination
	select {
	case <-quit:
		fmt.Println("quit")
		break
	case <-ctx.Done():
		fmt.Println("ctx done")
		break
	}

	// gracefully shutdown http servers
	timeoutCtx, timeoutCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer timeoutCancel()

	log.Println("shutting down servers, please wait...")

	server1.Shutdown(timeoutCtx)
	server2.Shutdown(timeoutCtx)

	// wait for shutdown
	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}

	log.Println("a graceful bye")
}
```





## Handling Errors

### 问题

> 我们在数据库操作的时候，比如 dao 层中当遇到一个 sql.ErrNoRows 的时候，是否应该 Wrap 这个 error，抛给上层。为什么，应该怎么做请写出代码？

### 解决

> 应该 Wrap 这个 error 抛给上层，由 biz 层处理这个 error，同时应该携带查询参数和堆栈信息，便于定位问题。

#### Handling Errors: Custom Error Type

```go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
)

// database handle
var db *sql.DB

type Album struct {
	ID     int64
	Title  string
	Artist string
	Price  float32
}

// AlbumNotFound Handling Errors: Error Types (Custom Error Types)
type AlbumNotFound struct {
	ID int64
}

func (e *AlbumNotFound) Error() string {
	return fmt.Sprintf("Album with ID %d not found", e.ID)
}

func main() {
	// Capture connection properties.
	cfg := mysql.Config{
		User:                 os.Getenv("DBUSER"),
		Passwd:               os.Getenv("DBPASS"),
		Net:                  "tcp",
		Addr:                 "127.0.0.1:3306",
		DBName:               "recordings",
		AllowNativePasswords: true,
	}

	// Get a database handle.
	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}
	
	alb, err = albumByID(10)

	// Handling Errors: Error Types (Custom Error Types)
	switch err := err.(type) {
	case nil:
    // success
		fmt.Printf("Album found: %v\n", alb)
	case *AlbumNotFound:
    // AlbumNotFound Error
		fmt.Printf("original error:\n%T %v\n", errors.Cause(err), errors.Cause(err))
		fmt.Printf("stack trace:\n%+v\n", err)
	default:
    // other Error
		fmt.Printf("original error:\n%T %v\n", errors.Cause(err), errors.Cause(err))
		fmt.Printf("stack trace:\n%+v\n", err)
	}
}

// queries for the album with the specified ID
func albumByID(id int64) (Album, error) {
	var alb Album

	// It returns an sql.Row. To simplify the calling code (your code!),
	row := db.QueryRow("SELECT * FROM album WHERE id = ?", id)
	// QueryRow doesn’t return an error. Instead, it arranges to return any query error
	// (such as sql.ErrNoRows) from Rows.Scan later.
	if err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
		// The special error sql.ErrNoRows indicates that the query returned no rows.
		// Typically that error is worth replacing with more specific text, such as “no such album” here.
		if err == sql.ErrNoRows {
			// Handling Errors: Error Types (Custom Error Types)
			return alb, errors.Wrapf(&AlbumNotFound{id}, fmt.Sprintf("albumById %d: no such album", id))
      // Handling Errors: Sentinel Error
			//return alb, errors.Wrapf(err, fmt.Sprintf("albumById %d: no such album", id)) 
		}
		return alb, errors.Wrapf(err, fmt.Sprintf("albumById %d: db query row error", id))
	}
	return alb, nil
}

```

#### Handling Errors: Opaque Errors

```go
// ...

// Handling Errors: Opaque errors
type albumNotFound interface {
	AlbumNotFound() (bool, int64)
}

func IsErrAlbumNotFound(err error) (bool, int64) {
	if e, ok := errors.Cause(err).(albumNotFound); ok {
		return e.AlbumNotFound()
	}
	return false, 0
}

type errAlbumNotFound struct {
	id int64
}

func (e *errAlbumNotFound) Error() string {
	return fmt.Sprintf("Album with Id %d not found", e.id)
}

func (e *errAlbumNotFound) AlbumNotFound() (bool, int64) {
	return true, e.id
}

func main() {
	// ...
  
	// Handling Errors: Opaque errors
	_, err = albumByID(10)
	if ok, _ := IsErrAlbumNotFound(err); ok {
    // errAlbumNotFound Error
		fmt.Printf("original error:\n%T %v\n", errors.Cause(err), errors.Cause(err))
		fmt.Printf("stack trace:\n%+v\n", err)
	}

	if err != nil {
    // other Error
    fmt.Printf("original error:\n%T %v\n", errors.Cause(err), errors.Cause(err))
		fmt.Printf("stack trace:\n%+v\n", err)
	}
  // success
  fmt.Printf("Album found: %v\n", alb)
}

// queries for the album with the specified ID
func albumByID(id int64) (Album, error) {
	var alb Album

	// It returns an sql.Row. To simplify the calling code (your code!),
	row := db.QueryRow("SELECT * FROM album WHERE id = ?", id)
	// QueryRow doesn’t return an error. Instead, it arranges to return any query error
	// (such as sql.ErrNoRows) from Rows.Scan later.
	if err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
		// The special error sql.ErrNoRows indicates that the query returned no rows.
		// Typically that error is worth replacing with more specific text, such as “no such album” here.
		if err == sql.ErrNoRows {
			return alb, errors.Wrapf(&errAlbumNotFound{id}, fmt.Sprintf("albumById %d: no such album", id))
		}
		return alb, errors.Wrapf(err, fmt.Sprintf("albumById %d: db query row error", id))
	}
	return alb, nil
}
```







