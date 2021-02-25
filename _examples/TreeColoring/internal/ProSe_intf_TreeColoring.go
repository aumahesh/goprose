// +build TreeColoring

package internal

import (
	"context"
)

type ProSe_intf_TreeColoring interface {
	EventHandler(context.Context)
	Listener(context.Context)
}

func NewProSe_intf_TreeColoring(id string, mcast string) (ProSe_intf_TreeColoring, error) {
	x := &ProSe_impl_TreeColoring{}
	err := x.init(id, mcast)
	if err != nil {
		return nil, err
	}
	return x, nil
}

