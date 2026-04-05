// +build dijkstra_gcl

package internal

import (
	"context"
)

type ProSe_intf_dijkstra_gcl interface {
	EventHandler(context.Context)
	Listener(context.Context)
}

func NewProSe_intf_dijkstra_gcl(id string, mcast string) (ProSe_intf_dijkstra_gcl, error) {
	x := &ProSe_impl_dijkstra_gcl{}
	err := x.init(id, mcast)
	if err != nil {
		return nil, err
	}
	return x, nil
}

