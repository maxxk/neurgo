
package neurgo

import (
)

type connection struct {
	other       Connector
	channel     VectorChannel
	weights     []float64
}

type Node struct {
	Name     string
	inbound  []*connection
	outbound []*connection
}

func (node *Node) String() string {
	return node.Name
}

func (node *Node) canPropagateSignal() bool {
	return len(node.inbound) > 0 
}

func (node *Node) propagateSignal() {
	panic("node.propagateSignal called")
}

func (node *Node) scatterOutput(outputs []float64) {
	for _, outboundConnection := range node.outbound {
		outboundConnection.channel <- outputs
	}
}

func (node *Node) ConnectBidirectional(target Connector) {
	node.ConnectBidirectionalWeighted(target, nil)
}

func (node *Node) ConnectBidirectionalWeighted(target Connector, weights []float64) {
	channel := make(VectorChannel)		
	node.connectOutboundWithChannel(target, channel)
	target.connectInboundWithChannel(node, channel, weights)
}

func (node *Node) connectOutboundWithChannel(target Connector, channel VectorChannel) {
	connection := &connection{channel: channel, other: target}
	node.outbound = append(node.outbound, connection)
}

func (node *Node) connectInboundWithChannel(source Connector, channel VectorChannel, weights []float64) {
	connection := &connection{channel: channel, weights: weights, other: source}
	node.inbound = append(node.inbound, connection)
}

func (node *Node) DisconnectBidirectional(target Connector) {
	node.disconnectOutbound(target)
	target.disconnectInbound(node)
}

func (node *Node) disconnectOutbound(target Connector) {
	for i, connection := range node.outbound {
		if connection.other == target {
			channel := node.outbound[i].channel
			node.outbound = removeConnection(node.outbound, i)
			close(channel)
		}
	}
}

func (node *Node) disconnectInbound(source Connector) {
	for i, connection := range node.inbound {
		if connection.other == source {
			node.inbound = removeConnection(node.inbound, i)
		}
	}
}

func (node *Node) outboundConnections() []*connection {
	return node.outbound
}

func (node *Node) inboundConnections() []*connection {
	return node.inbound
}

func (node *Node) appendOutboundConnection(target *connection) {
	node.outbound = append(node.outbound, target)
}

func (node *Node) appendInboundConnection(source *connection) {
	node.inbound = append(node.inbound, source)
}
	
