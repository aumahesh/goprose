// +build dijkstra_gcl_do

package internal

import (
	"context"
)

type ProSe_intf_dijkstra_gcl_do interface {
	EventHandler(context.Context)
	Listener(context.Context)
}

func NewProSe_intf_dijkstra_gcl_do(id string, mcast string) (ProSe_intf_dijkstra_gcl_do, error) {
	x := &ProSe_impl_dijkstra_gcl_do{}
	err := x.init(id, mcast)
	if err != nil {
		return nil, err
	}
	return x, nil
}

