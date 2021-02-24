// +build RoutingTreeMaintenance

package internal

import (
	"context"
)

type ProSe_intf_RoutingTreeMaintenance interface {
	EventHandler(context.Context)
	Listener(context.Context)
}

func NewProSe_intf_RoutingTreeMaintenance(id string, mcast string) (ProSe_intf_RoutingTreeMaintenance, error) {
	x := &ProSe_impl_RoutingTreeMaintenance{}
	err := x.init(id, mcast)
	if err != nil {
		return nil, err
	}
	return x, nil
}
