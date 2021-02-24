// +build RoutingTreeMaintenance

package internal

import (
	"context"
	"net"
	"time"


	"math/rand"


	log "github.com/sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"github.com/dmichael/go-multicast/multicast"
	p "aumahesh.com/prose/RoutingTreeMaintenance/models"
)

const (
	inactivityTimeout = time.Duration(2) * time.Minute
	heartbeatInterval = time.Duration(1) * time.Minute
	maxDatagramSize = 1024
)

var (
	
	
	CMAX int64 = 0
	
)

type NeighborState struct {
	id string
	state *p.State
	discoveredAt time.Time
	updatedAt time.Time
	lastHeartBeatAt time.Time
	stateChangedAt time.Time
	active bool
}

type RoutingTreeMaintenance_impl struct {
	id string
	state *p.State
	mcastAddr string
	mcastConn *net.UDPConn
	receiveChannel chan *p.NeighborUpdate
	hbChannel chan *p.NeighborHeartBeat
	neighborState map[string]*NeighborState
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
					id: this.id,
					state: this.state,
					discoveredAt: time.Now(),
					updatedAt: time.Now(),
					lastHeartBeatAt: time.Now(),
					stateChangedAt: time.Now(),
					active: true,
				},
	}

	CMAX = this.initConstantCMAX()
	

	this.initState()

	return nil
}

func (this *RoutingTreeMaintenance_impl) initState() {
	this.state.Dist = 0
        this.state.Inv = 0
        this.state.P = ""
        

	this.state.Dist = this.initVaribleDist()
	this.state.Inv = this.initVaribleInv()
	
}

func (this *RoutingTreeMaintenance_impl) initConstantCMAX() int64 {
	
	temp0 := int64(100)
	
	return temp0
}

func (this *RoutingTreeMaintenance_impl) initVaribleDist() int64 {
    
	temp2 := int64(10)
    
	temp3 := rand.Int63n(temp2)
    
	return temp3
}

func (this *RoutingTreeMaintenance_impl) initVaribleInv() int64 {
    
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
					id: s.Id,
					discoveredAt: time.Now(),
					active: true,
					stateChangedAt: time.Now(),
				}
			}
			this.neighborState[s.Id].state = &p.State{
				
					Dist: s.State.Dist,
					Inv: s.State.Inv,
					P: s.State.P,
			}
			this.neighborState[s.Id].updatedAt = time.Now()
			this.evaluateNeighborStates()
			this.updateLocalState()
			if stateChanged := this.updateLocalState(); stateChanged {
				this.broadcastLocalState()
				if err != nil {
					log.Errorf("Error broadcasting local state to neighbors")
				}
			}
		case s := <-this.hbChannel:
			_, ok := this.neighborState[s.Id]
			if !ok {
				this.neighborState[s.Id] = &NeighborState{
					id: s.Id,
					state: &p.State{},
					discoveredAt: time.Now(),
					active: true,
					stateChangedAt: time.Now(),
					updatedAt: time.Now(),
				}
			}
			this.neighborState[s.Id].lastHeartBeatAt = time.Now()
			this.evaluateNeighborStates()
			if stateChanged := this.updateLocalState(); stateChanged {
				this.broadcastLocalState()
				if err != nil {
					log.Errorf("Error broadcasting local state to neighbors")
				}
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

	log.Debugf("Executing: doAction0")

	
	var found bool
	var neighbor *NeighborState
	for _, neighbor = range this.neighborState {
		temp4 := neighbor.state.Dist
		temp5 := this.state.Dist
		temp6 := temp4 < temp5
		temp7 := neighbor.id
		temp8 := this.isNeighborUp(temp7)
		temp9 := neighbor.state.Inv
		temp10 := CMAX
		temp11 := temp9 < temp10
		temp12 := neighbor.state.Inv
		temp13 := this.state.Inv
		temp14 := temp12 < temp13
		temp15 := temp11 && temp14
		temp16 := temp8 && temp15
		temp17 := temp6 && temp16
		if temp17 {
			found = true
			break
		}
	}
	if found {
		temp18 := neighbor.id
		this.state.P = temp18
		temp19 := neighbor.state.Inv
		this.state.Inv = temp19
		stateChanged = true
	}

	log.Debugf("doAction0: state changed: %v", stateChanged)

	return stateChanged
}

func (this *RoutingTreeMaintenance_impl) doAction1() bool {
	stateChanged := false

	log.Debugf("Executing: doAction1")

	
	var found bool
	var neighbor *NeighborState
	for _, neighbor = range this.neighborState {
		temp20 := neighbor.state.Dist
		temp21 := this.state.Dist
		temp22 := temp20 < temp21
		temp23 := neighbor.id
		temp24 := this.isNeighborUp(temp23)
		temp25 := neighbor.state.Inv
		temp26 := int64(1)
		temp27 := temp25 + temp26
		temp28 := CMAX
		temp29 := temp27 < temp28
		temp30 := neighbor.state.Inv
		temp31 := int64(1)
		temp32 := temp30 + temp31
		temp33 := this.state.Inv
		temp34 := temp32 < temp33
		temp35 := temp29 && temp34
		temp36 := temp24 && temp35
		temp37 := temp22 && temp36
		if temp37 {
			found = true
			break
		}
	}
	if found {
		temp38 := neighbor.id
		this.state.P = temp38
		temp39 := neighbor.state.Inv
		temp40 := int64(1)
		temp41 := temp39 + temp40
		this.state.Inv = temp41
		stateChanged = true
	}

	log.Debugf("doAction1: state changed: %v", stateChanged)

	return stateChanged
}

func (this *RoutingTreeMaintenance_impl) doAction2() bool {
	stateChanged := false

	log.Debugf("Executing: doAction2")

	
	var found bool
	var neighbor *NeighborState
	for _, neighbor = range this.neighborState {
		temp42 := this.state.P
		temp43 := ""
		temp44 := temp42 != temp43
		temp45 := this.state.P
		temp46 := this.isNeighborUp(temp45)
		temp47 := false
		temp48 := temp46 == temp47
		var temp49 int64
		if neighbor.id == this.state.P {
			temp49 = this.state.Inv
		} else {
			continue
		}
		temp50 := CMAX
		temp51 := temp49 >= temp50
		var temp52 int64
		if neighbor.id == this.state.P {
			temp52 = this.state.Dist
		} else {
			continue
		}
		temp53 := this.state.Dist
		temp54 := temp52 < temp53
		temp55 := this.state.Inv
		var temp56 int64
		if neighbor.id == this.state.P {
			temp56 = this.state.Inv
		} else {
			continue
		}
		temp57 := temp55 != temp56
		temp58 := temp54 && temp57
		var temp59 int64
		if neighbor.id == this.state.P {
			temp59 = this.state.Dist
		} else {
			continue
		}
		temp60 := this.state.Dist
		temp61 := temp59 > temp60
		temp62 := this.state.Inv
		var temp63 int64
		if neighbor.id == this.state.P {
			temp63 = this.state.Inv
		} else {
			continue
		}
		temp64 := int64(1)
		temp65 := temp63 + temp64
		temp66 := temp62 != temp65
		temp67 := temp61 && temp66
		temp68 := temp58 || temp67
		temp69 := temp51 || temp68
		temp70 := temp48 || temp69
		temp71 := temp44 && temp70
		if temp71 {
			found = true
			break
		}
	}
	if found {
		temp72 := ""
		this.state.P = temp72
		temp73 := CMAX
		this.state.Inv = temp73
		stateChanged = true
	}

	log.Debugf("doAction2: state changed: %v", stateChanged)

	return stateChanged
}

func (this *RoutingTreeMaintenance_impl) doAction3() bool {
	stateChanged := false

	log.Debugf("Executing: doAction3")

	
	temp74 := this.state.P
	temp75 := ""
	temp76 := temp74 == temp75
	temp77 := this.state.Inv
	temp78 := CMAX
	temp79 := temp77 < temp78
	temp80 := temp76 && temp79
	if temp80 {
		temp81 := CMAX
		this.state.Inv = temp81
		stateChanged = true
	}

	log.Debugf("doAction3: state changed: %v", stateChanged)

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
                Inv: this.state.Inv,
                P: this.state.P,
		},
	}
	broadcastMessage := &p.BroadcastMessage{
		Type: p.MessageType_StateUpdate,
		Src: this.id,
		Msg:  &p.BroadcastMessage_Upd{updMessage},
	}

	return this.send(broadcastMessage)
}

func (this *RoutingTreeMaintenance_impl) sendHeartBeat() (int, error) {
	hbMessage := &p.NeighborHeartBeat{
		Id: this.id,
		SentAt: time.Now().Unix(),
	}
	broadcastMessage := &p.BroadcastMessage{
		Type: p.MessageType_Heartbeat,
		Src: this.id,
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