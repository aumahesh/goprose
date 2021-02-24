// +build DistributedReset

package internal

import (
	"context"
)

type ProSe_intf_DistributedReset interface {
	EventHandler(context.Context)
	Listener(context.Context)
}

func NewProSe_intf_DistributedReset(id string, mcast string) (ProSe_intf_DistributedReset, error) {
	x := &ProSe_impl_DistributedReset{}
	err := x.init(id, mcast)
	if err != nil {
		return nil, err
	}
	return x, nil
}
