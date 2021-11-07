# learn-go

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







