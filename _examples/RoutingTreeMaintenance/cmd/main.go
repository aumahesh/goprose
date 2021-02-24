// +build RoutingTreeMaintenance

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/google/uuid"
	"aumahesh.com/prose/RoutingTreeMaintenance/internal"
)

const (
	defaultMcastAddress = "239.0.0.0:9999"
)

func main() {
	id := uuid.Must(uuid.NewRandom()).String()

	log.SetLevel(log.DebugLevel)
	log.Debugf("%s: Hello, world!", id)

	impl, err := internal.NewRoutingTreeMaintenance_intf(id, defaultMcastAddress)
	if err != nil {
		log.Errorf("error instantiating RoutingTreeMaintenance_intf: %s", err)
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	go impl.EventHandler(ctx)
	go impl.Listener(ctx)

	<-signalCh
}