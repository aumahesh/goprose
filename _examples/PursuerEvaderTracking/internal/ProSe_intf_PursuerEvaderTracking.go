// +build PursuerEvaderTracking

package internal

import (
	"context"
)

type ProSe_intf_PursuerEvaderTracking interface {
	EventHandler(context.Context)
	Listener(context.Context)
}

func NewProSe_intf_PursuerEvaderTracking(id string, mcast string) (ProSe_intf_PursuerEvaderTracking, error) {
	x := &ProSe_impl_PursuerEvaderTracking{}
	err := x.init(id, mcast)
	if err != nil {
		return nil, err
	}
	return x, nil
}

