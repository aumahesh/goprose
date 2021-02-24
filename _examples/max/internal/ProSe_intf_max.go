// +build max

package internal

import (
	"context"
)

type ProSe_intf_max interface {
	EventHandler(context.Context)
	Listener(context.Context)
}

func NewProSe_intf_max(id string, mcast string) (ProSe_intf_max, error) {
	x := &ProSe_impl_max{}
	err := x.init(id, mcast)
	if err != nil {
		return nil, err
	}
	return x, nil
}
