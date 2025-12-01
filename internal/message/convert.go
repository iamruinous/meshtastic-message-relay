// Package message defines message types and conversion utilities.
package message

import (
	"time"

	"github.com/iamruinous/meshtastic-message-relay/pkg/meshtastic"
)

// FromMeshtasticPacket converts a meshtastic.Packet to our internal Packet format
func FromMeshtasticPacket(mp *meshtastic.Packet) *Packet {
	if mp == nil {
		return nil
	}

	p := &Packet{
		ID:         mp.ID,
		From:       mp.From,
		To:         mp.To,
		Channel:    mp.Channel,
		PortNum:    PortNum(mp.PortNum),
		RawPayload: mp.RawPayload,
		SNR:        mp.SNR,
		RSSI:       mp.RSSI,
		HopLimit:   mp.HopLimit,
		WantAck:    mp.WantAck,
		ReceivedAt: mp.ReceivedAt,
	}

	// Convert payload
	switch payload := mp.Payload.(type) {
	case *meshtastic.TextMessage:
		p.Payload = &TextMessage{Text: payload.Text}
	case *meshtastic.Position:
		p.Payload = &Position{
			Latitude:  payload.Latitude(),
			Longitude: payload.Longitude(),
			Altitude:  payload.Altitude,
			Time:      time.Unix(int64(payload.Time), 0),
		}
	default:
		p.Payload = payload
	}

	// Convert node info if present
	if mp.FromNode != nil {
		p.FromNode = FromMeshtasticNodeInfo(mp.FromNode)
	}

	return p
}

// FromMeshtasticNodeInfo converts a meshtastic.NodeInfo to our internal NodeInfo format
func FromMeshtasticNodeInfo(mn *meshtastic.NodeInfo) *NodeInfo {
	if mn == nil {
		return nil
	}

	ni := &NodeInfo{
		Num:       mn.Num,
		SNR:       mn.Snr,
		LastHeard: time.Unix(int64(mn.LastHeard), 0),
	}

	if mn.User != nil {
		ni.User = &User{
			ID:        mn.User.ID,
			LongName:  mn.User.LongName,
			ShortName: mn.User.ShortName,
		}
	}

	if mn.Position != nil {
		ni.Position = &Position{
			Latitude:  mn.Position.Latitude(),
			Longitude: mn.Position.Longitude(),
			Altitude:  mn.Position.Altitude,
			Time:      time.Unix(int64(mn.Position.Time), 0),
		}
	}

	return ni
}
