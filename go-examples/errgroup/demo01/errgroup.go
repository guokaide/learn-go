package main

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func hello(w http.ResponseWriter, req *http.Request)  {
	fmt.Fprintf(w, "hello, world\n")
}

func startHttpServer(server *http.Server) error {
	http.HandleFunc("/hello", hello)
	return server.ListenAndServe()
}

func main() {
	g, ctx := errgroup.WithContext(context.Background())

	server := &http.Server{Addr: ":9093"}

	// Start http server
	g.Go(func() error {
		fmt.Println("http server")
		go func() {
			// Why?
			<- ctx.Done()
			fmt.Println("http ctx down")
			server.Shutdown(ctx)
		}()
		return startHttpServer(server)
	})

	// Awaiting signals
	g.Go(func() error {
		exitSignals := []os.Signal{
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT,
			os.Interrupt,
		}
		sigs := make(chan os.Signal, len(exitSignals))
		// Registers the given channel to receive notifications of
		// the specified signals.
		// signal -> sigs channel
		signal.Notify(sigs, exitSignals...)
		for {
			fmt.Println("awaiting signal")
			select {
			case <- ctx.Done():
				fmt.Println("signal ctx done")
				return ctx.Err()
			case <- sigs:
				fmt.Println("receive exit signal")
				return errors.New("bye")
			}
		}
	})

	err := g.Wait()
	fmt.Println(err)
}
