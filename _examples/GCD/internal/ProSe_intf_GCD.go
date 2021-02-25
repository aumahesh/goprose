// +build GCD

package internal

import (
	"context"
)

type ProSe_intf_GCD interface {
	EventHandler(context.Context)
	Listener(context.Context)
}

func NewProSe_intf_GCD(id string, mcast string) (ProSe_intf_GCD, error) {
	x := &ProSe_impl_GCD{}
	err := x.init(id, mcast)
	if err != nil {
		return nil, err
	}
	return x, nil
}

