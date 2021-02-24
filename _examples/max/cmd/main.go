// +build max

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"aumahesh.com/prose/max/internal"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

const (
	defaultMcastAddress = "239.0.0.0:9999"
)

func main() {
	id := uuid.Must(uuid.NewRandom()).String()

	log.SetLevel(log.DebugLevel)
	log.Debugf("%s: Hello, world!", id)

	impl, err := internal.NewProSe_intf_max(id, defaultMcastAddress)
	if err != nil {
		log.Errorf("error instantiating ProSe_intf_max: %s", err)
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
