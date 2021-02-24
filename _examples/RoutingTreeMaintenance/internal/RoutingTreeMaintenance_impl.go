// +build RoutingTreeMaintenance

package internal

import (
	"context"
	"net"
	"time"

	p "aumahesh.com/prose/RoutingTreeMaintenance/models"
	"github.com/dmichael/go-multicast/multicast"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
)

const (
	inactivityTimeout = time.Duration(2) * time.Minute
	heartbeatInterval = time.Duration(10) * time.Second
	maxDatagramSize   = 1024
)

var (
	CMAX int64 = 0

	NULL int64 = 0
)

type NeighborState struct {
	id              string
	state           *p.State
	discoveredAt    time.Time
	updatedAt       time.Time
	lastHeartBeatAt time.Time
	stateChangedAt  time.Time
	active          bool
}

type RoutingTreeMaintenance_impl struct {
	id             string
	state          *p.State
	mcastAddr      string
	mcastConn      *net.UDPConn
	receiveChannel chan *p.NeighborUpdate
	hbChannel      chan *p.NeighborHeartBeat
	neighborState  map[string]*NeighborState
}

func (this *RoutingTreeMaintenance_impl) init(id string, mcastAddr string) error {
	this.id = id
	this.state = &p.State{}
	this.mcastAddr = mcastAddr

	conn, err := multicast.NewBroadcaster(this.mcastAddr)
	if err != nil {
		return err
	}
	this.mcastConn = conn

	this.receiveChannel = make(chan *p.NeighborUpdate, 10)
	this.hbChannel = make(chan *p.NeighborHeartBeat, 10)

	this.neighborState = map[string]*NeighborState{
		this.id: &NeighborState{
			id:              this.id,
			state:           this.state,
			discoveredAt:    time.Now(),
			updatedAt:       time.Now(),
			lastHeartBeatAt: time.Now(),
			stateChangedAt:  time.Now(),
			active:          true,
		},
	}

	CMAX = this.initConstantCMAX()
	NULL = this.initConstantNULL()

	this.initState()

	return nil
}

func (this *RoutingTreeMaintenance_impl) initState() {
	this.state.Dist = 0
	this.state.Inv = 0
	this.state.P = ""
	this.state.Up = false

}

func (this *RoutingTreeMaintenance_impl) initConstantCMAX() int64 {

	temp0 := int64(100)

	return temp0
}

func (this *RoutingTreeMaintenance_impl) initConstantNULL() int64 {

	temp1 := int64(0)

	return temp1
}

func (this *RoutingTreeMaintenance_impl) EventHandler(ctx context.Context) {
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	for {
		select {
		case s := <-this.receiveChannel:
			_, ok := this.neighborState[s.Id]
			if !ok {
				this.neighborState[s.Id] = &NeighborState{
					id:             s.Id,
					discoveredAt:   time.Now(),
					active:         true,
					stateChangedAt: time.Now(),
				}
			}
			this.neighborState[s.Id].state = &p.State{

				Dist: s.State.Dist,
				Inv:  s.State.Inv,
				P:    s.State.P,
				Up:   s.State.Up,
			}
			this.neighborState[s.Id].updatedAt = time.Now()
			this.evaluateNeighborStates()
			this.updateLocalState()
			_, err := this.broadcastLocalState()
			if err != nil {
				log.Errorf("Error broadcasting local state to neighbors")
			}
		case s := <-this.hbChannel:
			_, ok := this.neighborState[s.Id]
			if !ok {
				this.neighborState[s.Id] = &NeighborState{
					id:             s.Id,
					discoveredAt:   time.Now(),
					active:         true,
					stateChangedAt: time.Now(),
					updatedAt:      time.Now(),
				}
			}
			this.neighborState[s.Id].lastHeartBeatAt = time.Now()
			this.evaluateNeighborStates()
			if stateChanged := this.updateLocalState(); stateChanged {
				this.broadcastLocalState()
			}
		case <-heartbeatTicker.C:
			this.sendHeartBeat()
		case <-ctx.Done():
			return
		}
	}
}

func (this *RoutingTreeMaintenance_impl) evaluateNeighborStates() {
	for id, nbr := range this.neighborState {
		if nbr.updatedAt.Add(inactivityTimeout).Before(time.Now()) {
			nbr.active = false
			nbr.stateChangedAt = time.Now()
			log.Warnf("neighbor %s became inactive at %s", id, time.Now())
		} else {
			if !nbr.active {
				log.Infof("neighbor %s became active at %s", id, time.Now())
			}
			nbr.active = true
			nbr.stateChangedAt = time.Now()
		}
		log.Debugf("state: %s: (%v) -> %+v", id, nbr.active, nbr.state)
	}
}

func (this *RoutingTreeMaintenance_impl) isNeighborUp(id string) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	return nbr.active
}

func (this *RoutingTreeMaintenance_impl) neighbors() map[string]*NeighborState {
	return this.neighborState
}

func (this *RoutingTreeMaintenance_impl) setNeighbor(id string, state bool) bool {
	nbr, ok := this.neighborState[id]
	if !ok {
		return false
	}
	nbr.active = state
	return nbr.active
}

func (this *RoutingTreeMaintenance_impl) doAction0() bool {
	stateChanged := false

	var found bool
	var neighbor *NeighborState
	for _, neighbor = range this.neighborState {
		temp2 := neighbor.state.Dist
		temp3 := this.state.Dist
		temp4 := temp2 < temp3
		temp5 := neighbor.id
		temp6 := this.isNeighborUp(temp5)
		temp7 := neighbor.state.Inv
		temp8 := CMAX
		temp9 := temp7 < temp8
		temp10 := neighbor.state.Inv
		temp11 := this.state.Inv
		temp12 := temp10 < temp11
		temp13 := temp9 && temp12
		temp14 := temp6 && temp13
		temp15 := temp4 && temp14
		if temp15 {
			found = true
			break
		}
	}
	if found {
		temp16 := neighbor.id
		this.state.P = temp16
		temp17 := neighbor.state.Inv
		this.state.Inv = temp17
		stateChanged = true
	}

	return stateChanged
}

func (this *RoutingTreeMaintenance_impl) doAction1() bool {
	stateChanged := false

	var found bool
	var neighbor *NeighborState
	for _, neighbor = range this.neighborState {
		temp18 := neighbor.state.Dist
		temp19 := this.state.Dist
		temp20 := temp18 < temp19
		temp21 := neighbor.id
		temp22 := this.isNeighborUp(temp21)
		temp23 := neighbor.state.Inv
		temp24 := int64(1)
		temp25 := temp23 + temp24
		temp26 := CMAX
		temp27 := temp25 < temp26
		temp28 := neighbor.state.Inv
		temp29 := int64(1)
		temp30 := temp28 + temp29
		temp31 := this.state.Inv
		temp32 := temp30 < temp31
		temp33 := temp27 && temp32
		temp34 := temp22 && temp33
		temp35 := temp20 && temp34
		if temp35 {
			found = true
			break
		}
	}
	if found {
		temp36 := neighbor.id
		this.state.P = temp36
		temp37 := neighbor.state.Inv
		temp38 := int64(1)
		temp39 := temp37 + temp38
		this.state.Inv = temp39
		stateChanged = true
	}

	return stateChanged
}

func (this *RoutingTreeMaintenance_impl) doAction2() bool {
	stateChanged := false

	var found bool
	var neighbor *NeighborState
	for _, neighbor = range this.neighborState {
		temp40 := this.state.P
		temp41 := ""
		temp42 := temp40 != temp41
		temp43 := this.state.P
		temp44 := this.isNeighborUp(temp43)
		temp45 := false
		temp46 := temp44 == temp45
		var temp47 int64
		if neighbor.id == this.state.P {
			temp47 = this.state.Inv
		} else {
			continue
		}
		temp48 := CMAX
		temp49 := temp47 >= temp48
		var temp50 int64
		if neighbor.id == this.state.P {
			temp50 = this.state.Dist
		} else {
			continue
		}
		temp51 := this.state.Dist
		temp52 := temp50 < temp51
		temp53 := this.state.Inv
		var temp54 int64
		if neighbor.id == this.state.P {
			temp54 = this.state.Inv
		} else {
			continue
		}
		temp55 := temp53 != temp54
		temp56 := temp52 && temp55
		var temp57 int64
		if neighbor.id == this.state.P {
			temp57 = this.state.Dist
		} else {
			continue
		}
		temp58 := this.state.Dist
		temp59 := temp57 > temp58
		temp60 := this.state.Inv
		var temp61 int64
		if neighbor.id == this.state.P {
			temp61 = this.state.Inv
		} else {
			continue
		}
		temp62 := int64(1)
		temp63 := temp61 + temp62
		temp64 := temp60 != temp63
		temp65 := temp59 && temp64
		temp66 := temp56 || temp65
		temp67 := temp49 || temp66
		temp68 := temp46 || temp67
		temp69 := temp42 && temp68
		if temp69 {
			found = true
			break
		}
	}
	if found {
		temp70 := ""
		this.state.P = temp70
		temp71 := CMAX
		this.state.Inv = temp71
		stateChanged = true
	}

	return stateChanged
}

func (this *RoutingTreeMaintenance_impl) doAction3() bool {
	stateChanged := false

	temp72 := this.state.P
	temp73 := ""
	temp74 := temp72 == temp73
	temp75 := this.state.Inv
	temp76 := CMAX
	temp77 := temp75 < temp76
	temp78 := temp74 && temp77
	if temp78 {
		temp79 := CMAX
		this.state.Inv = temp79
		stateChanged = true
	}

	return stateChanged
}

func (this *RoutingTreeMaintenance_impl) updateLocalState() bool {
	stateChanged := false

	statements := []func() bool{

		this.doAction0,

		this.doAction1,

		this.doAction2,

		this.doAction3,
	}

	for _, stmtFunc := range statements {
		if changed := stmtFunc(); changed {
			stateChanged = true
		}
	}

	return stateChanged
}

func (this *RoutingTreeMaintenance_impl) broadcastLocalState() (int, error) {
	updMessage := &p.NeighborUpdate{
		Id: this.id,
		State: &p.State{

			Dist: this.state.Dist,
			Inv:  this.state.Inv,
			P:    this.state.P,
			Up:   this.state.Up,
		},
	}
	broadcastMessage := &p.BroadcastMessage{
		Type: p.MessageType_StateUpdate,
		Src:  this.id,
		Msg:  &p.BroadcastMessage_Upd{updMessage},
	}

	return this.send(broadcastMessage)
}

func (this *RoutingTreeMaintenance_impl) sendHeartBeat() (int, error) {
	hbMessage := &p.NeighborHeartBeat{
		Id:     this.id,
		SentAt: time.Now().Unix(),
	}
	broadcastMessage := &p.BroadcastMessage{
		Type: p.MessageType_Heartbeat,
		Src:  this.id,
		Msg:  &p.BroadcastMessage_Hb{hbMessage},
	}

	return this.send(broadcastMessage)
}

func (this *RoutingTreeMaintenance_impl) send(msg *p.BroadcastMessage) (int, error) {
	log.Debugf("Sending: %+v", msg)
	data, err := proto.Marshal(msg)
	if err != nil {
		return 0, err
	}
	return this.mcastConn.Write(data)
}

func (this *RoutingTreeMaintenance_impl) msgHandler(src *net.UDPAddr, n int, b []byte) {
	log.Debugf("received message (%d bytes) from %s", n, src.String())
	broadcastMessage := &p.BroadcastMessage{}
	err := proto.Unmarshal(b[:n], broadcastMessage)
	if err != nil {
		log.Errorf("error unmarshalling proto from src %s: %s", src.String(), err)
		return
	}
	log.Debugf("received: %+v", broadcastMessage)
	switch broadcastMessage.Type {
	case p.MessageType_Heartbeat:
		this.hbChannel <- broadcastMessage.GetHb()
	case p.MessageType_StateUpdate:
		this.receiveChannel <- broadcastMessage.GetUpd()
	default:
		log.Errorf("invalid message type")
	}
}

func (this *RoutingTreeMaintenance_impl) Listener(ctx context.Context) {
	addr, err := net.ResolveUDPAddr("udp4", this.mcastAddr)
	if err != nil {
		log.Errorf("Error resolving mcast address: %s", err)
		return
	}

	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		log.Errorf("Error connecting to mcast address: %s", err)
	}

	conn.SetReadBuffer(maxDatagramSize)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			buffer := make([]byte, maxDatagramSize)
			numBytes, src, err := conn.ReadFromUDP(buffer)
			if err != nil {
				log.Fatal("ReadFromUDP failed:", err)
			}
			this.msgHandler(src, numBytes, buffer)
		}
	}
}
