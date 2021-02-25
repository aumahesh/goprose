// +build example

package internal

import (
	"context"
)

type ProSe_intf_example interface {
	EventHandler(context.Context)
	Listener(context.Context)
}

func NewProSe_intf_example(id string, mcast string) (ProSe_intf_example, error) {
	x := &ProSe_impl_example{}
	err := x.init(id, mcast)
	if err != nil {
		return nil, err
	}
	return x, nil
}

