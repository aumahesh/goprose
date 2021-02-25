// +build pCover

package internal

import (
	"context"
)

type ProSe_intf_pCover interface {
	EventHandler(context.Context)
	Listener(context.Context)
}

func NewProSe_intf_pCover(id string, mcast string) (ProSe_intf_pCover, error) {
	x := &ProSe_impl_pCover{}
	err := x.init(id, mcast)
	if err != nil {
		return nil, err
	}
	return x, nil
}

