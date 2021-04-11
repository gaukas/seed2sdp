package seed2sdp

import (
	"fmt"
	"hash/crc32"
	"net"

	"github.com/pion/ice/v2"
)

const (
	defaultLocalPreference uint16 = 65535
)

type ICECandidate struct {
	foundation    uint32             // Calculated
	component     ICEComponent       // RTP: 1, RTCP: 2
	protocol      ICENetworkProtocol // tcp/udp
	priority      uint32             // Calculated
	ipAddr        net.IP
	port          uint16
	candidateType ICECandidateType // host, srflx, prflx, relay, unknown
	tcpType       ice.TCPType      // TCPTypeActive, TCPTypePassive, TCPTypeSimultaneousOpen, TCPTypeUnspecified
}

func (ic ICECandidateType) String() string {
	switch ic {
	case Host:
		return "host"
	case Srflx:
		return "srflx"
	case Prflx:
		return "prflx"
	case Relay:
		return "relay"
	default:
		return "unknown"
	}
}

func (c *ICECandidate) NetworkType() ICENetworkType {
	if c.ipAddr.To4() != nil {
		if c.protocol == UDP {
			return UDP4
		} else if c.protocol == TCP {
			return TCP4
		}
	} else if c.ipAddr.To16() != nil {
		if c.protocol == UDP {
			return UDP6
		} else if c.protocol == TCP {
			return TCP6
		}
	}
	return BADNETWORKTYPE
}

func (c *ICECandidate) Foundation() uint32 {
	if c.foundation == 0 { // If not set, set the foundation.
		c.foundation = crc32.ChecksumIEEE([]byte(c.candidateType.String() + c.ipAddr.String() + c.NetworkType().String()))
	}

	return c.foundation
}

func (c *ICECandidate) Preference() uint32 {
	switch t := c.candidateType; t {
	case Host:
		return 126
	case Srflx:
		return 100
	case Prflx:
		return 110
	case Relay:
		return 0
	default:
		return 0
	}
}

func (c *ICECandidate) LocalPreference() uint16 {
	if c.protocol == TCP {
		var otherPref uint16 = 8191

		directionPref := func() uint16 {
			switch c.candidateType {
			case Host, Relay:
				switch c.tcpType {
				case ice.TCPTypeActive:
					return 6
				case ice.TCPTypePassive:
					return 4
				case ice.TCPTypeSimultaneousOpen:
					return 2
				case ice.TCPTypeUnspecified:
					return 0
				}
			case Prflx, Srflx:
				switch c.tcpType {
				case ice.TCPTypeSimultaneousOpen:
					return 6
				case ice.TCPTypeActive:
					return 4
				case ice.TCPTypePassive:
					return 2
				case ice.TCPTypeUnspecified:
					return 0
				}
			case Unknown:
				return 0
			}
			return 0
		}()
		return (1<<13)*directionPref + otherPref
	}
	return defaultLocalPreference
}

func (c *ICECandidate) OffsetComponentPreference() uint32 {
	return uint32(256 - uint32(c.component))
}

func (c *ICECandidate) Priority() uint32 {
	if c.priority == 0 { // If not set, set the priority.
		c.priority = (1<<24)*c.Preference() + (1<<8)*uint32(c.LocalPreference()) + c.OffsetComponentPreference()
	}
	return c.priority
}

func (c *ICECandidate) SetFoundation(foundation uint32) *ICECandidate {
	c.foundation = foundation
	return c
}

// Necessary
func (c *ICECandidate) SetComponent(component ICEComponent) *ICECandidate {
	switch component {
	case ICEComponentRTP:
		c.component = ICEComponentRTP
	case ICEComponentRTCP:
		c.component = ICEComponentRTCP
	default:
		c.component = ICEComponentUnknown
	}
	return c
}

func (c *ICECandidate) SetProtocol(protocol ICENetworkProtocol) *ICECandidate {
	switch protocol {
	case UDP:
		c.protocol = UDP
	case TCP:
		c.protocol = TCP
	default:
		c.protocol = BADNETWORKPROTOCOL
	}
	return c
}

func (c *ICECandidate) SetPriority(priority uint32) *ICECandidate {
	c.priority = priority
	return c
}

func (c *ICECandidate) SetIpAddr(ipAddr net.IP) *ICECandidate {
	if ipAddr.To16() == nil {
		c.ipAddr = nil
	} else {
		c.ipAddr = ipAddr
	}
	return c
}

func (c *ICECandidate) SetPort(port uint16) *ICECandidate {
	c.port = port
	return c
}

func (c *ICECandidate) SetCandidateType(candidateType ICECandidateType) *ICECandidate {
	c.candidateType = candidateType
	return c
}

func (c *ICECandidate) SetTcpType(tcpType ice.TCPType) {
	c.tcpType = tcpType
}

func candidateToString(c ICECandidate) string {
	parsed_c := fmt.Sprintf(`a=candidate:%d %d %s %d %s %d typ %s`,
		c.Foundation(),
		c.component,
		c.protocol.String(),
		c.Priority(),
		c.ipAddr.String(),
		c.port,
		c.candidateType.String())

	// TODO: TCP: Append TcpType

	parsed_c += `\r\n`

	return parsed_c
}

func candidatesToString(arrc []ICECandidate) string {
	parsed_as := ""

	for _, c := range arrc {
		parsed_as += candidateToString(c)
	}

	parsed_as += `a=end-of-candidates\r\n`
	return parsed_as
}
