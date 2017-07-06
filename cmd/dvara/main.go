package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pataquets/dvara"
	"github.com/facebookgo/inject"
	"github.com/facebookgo/startstop"
	"github.com/facebookgo/stats"
)

func main() {
	if err := Main(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Main() error {
	messageTimeout := flag.Duration("message_timeout", 2*time.Minute, "timeout for one message to be proxied")
	clientIdleTimeout := flag.Duration("client_idle_timeout", 60*time.Minute, "idle timeout for client connections")
	serverIdleTimeout := flag.Duration("server_idle_timeout", 1*time.Hour, "idle timeout for  server connections")
	serverClosePoolSize := flag.Uint("server_close_pool_size", 100, "number of goroutines that will handle closing server connections")
	getLastErrorTimeout := flag.Duration("get_last_error_timeout", time.Minute, "timeout for getLastError pinning")
	maxPerClientConnections := flag.Uint("max_per_client_connections", 100, "maximum number of connections per client")
	maxConnections := flag.Uint("max_connections", 100, "maximum number of connections per mongo")
	portStart := flag.Int("port_start", 6000, "start of port range")
	portEnd := flag.Int("port_end", 6010, "end of port range")
	addrs := flag.String("addrs", "mongossl:27017", "comma separated list of mongo addresses")

	flag.Parse()

	replicaSet := dvara.ReplicaSet{
		Addrs:                   *addrs,
		PortStart:               *portStart,
		PortEnd:                 *portEnd,
		MessageTimeout:          *messageTimeout,
		ClientIdleTimeout:       *clientIdleTimeout,
		ServerIdleTimeout:       *serverIdleTimeout,
		ServerClosePoolSize:     *serverClosePoolSize,
		GetLastErrorTimeout:     *getLastErrorTimeout,
		MaxConnections:          *maxConnections,
		MaxPerClientConnections: *maxPerClientConnections,
	}

	var statsClient stats.HookClient
	var log stdLogger
	var graph inject.Graph
	err := graph.Provide(
		&inject.Object{Value: &log},
		&inject.Object{Value: &replicaSet},
		&inject.Object{Value: &statsClient},
	)
	if err != nil {
		return err
	}
	if err := graph.Populate(); err != nil {
		return err
	}
	objects := graph.Objects()

	if err := startstop.Start(objects, &log); err != nil {
		return err
	}
	defer startstop.Stop(objects, &log)

	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	<-ch
	signal.Stop(ch)
	return nil
}
