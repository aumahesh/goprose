// +build RoutingTreeMaintenance

package internal

import (
	"context"
)

type RoutingTreeMaintenance_intf interface {
	EventHandler(context.Context)
	Listener(context.Context)
}

func NewRoutingTreeMaintenance_intf(id string, mcast string) (RoutingTreeMaintenance_intf, error) {
	x := &RoutingTreeMaintenance_impl{}
	err := x.init(id, mcast)
	if err != nil {
		return nil, err
	}
	return x, nil
}

