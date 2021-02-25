// +build TrackingPriority

package internal

import (
	"context"
)

type ProSe_intf_TrackingPriority interface {
	EventHandler(context.Context)
	Listener(context.Context)
}

func NewProSe_intf_TrackingPriority(id string, mcast string) (ProSe_intf_TrackingPriority, error) {
	x := &ProSe_impl_TrackingPriority{}
	err := x.init(id, mcast)
	if err != nil {
		return nil, err
	}
	return x, nil
}

