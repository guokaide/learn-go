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