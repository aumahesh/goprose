// +build distanceVector

package internal

import (
	"context"
)

type ProSe_intf_distanceVector interface {
	EventHandler(context.Context)
	Listener(context.Context)
}

func NewProSe_intf_distanceVector(id string, mcast string) (ProSe_intf_distanceVector, error) {
	x := &ProSe_impl_distanceVector{}
	err := x.init(id, mcast)
	if err != nil {
		return nil, err
	}
	return x, nil
}

