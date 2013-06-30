package neurgo

import (
	"encoding/json"
	"fmt"
	"sync"
)

type NeuralNetwork struct {
	sensors   []*Node
	actuators []*Node
	Node
}

type copyScaffold struct {
	nodeScaffold    map[*Node]*Node
	channelScaffold map[VectorChannel]VectorChannel
}

const FITNESS_THRESHOLD = 1e8

type NodeMap map[*Node]*Node

func (neuralNet *NeuralNetwork) Fitness(samples []*TrainingSample) float64 {

	errorAccumulated := float64(0)

	// make as many injectors as there are sensors
	injectors := make([]*Node, len(neuralNet.sensors))
	for i, _ := range injectors {
		injectors[i] = &Node{}
		injectors[i].setInvisible(true)
		injectors[i].Name = fmt.Sprintf("injector-%d", i+1)
		injectors[i].ConnectBidirectional(neuralNet.sensors[i])
	}

	// make as many wiretaps as actuators
	wiretaps := make([]*Node, len(neuralNet.actuators))
	for i, _ := range wiretaps {
		wiretaps[i] = &Node{}
		wiretaps[i].setInvisible(true)
		wiretaps[i].Name = fmt.Sprintf("wiretap-%d", i+1)
		neuralNet.actuators[i].ConnectBidirectional(wiretaps[i])
	}

	neuralNet.Run()

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Add(1)

	// inject values into sensors
	go func() {
		for _, sample := range samples {
			for j, inputsForSensor := range sample.SampleInputs {
				injectors[j].outbound[0].channel <- inputsForSensor
			}
		}
		wg.Done()
	}()

	// read the value from wiretap (which taps into actuator)
	go func() {

		for _, sample := range samples {
			for j, expectedOutputs := range sample.ExpectedOutputs {
				resultVector := <-wiretaps[j].inbound[0].channel
				error := SumOfSquaresError(expectedOutputs, resultVector)
				errorAccumulated += error
			}
		}

		wg.Done()
	}()

	wg.Wait()

	neuralNet.Shutdown()

	// disconnect injectors and wiretaps to leave it in the same state!
	for i, injector := range injectors {
		injector.DisconnectBidirectional(neuralNet.sensors[i])
	}
	for i, actuator := range neuralNet.actuators {
		actuator.DisconnectBidirectional(wiretaps[i])
	}

	return float64(1) / errorAccumulated

}

// Make sure the neural network gives expected output for the given
// training samples.
func (neuralNet *NeuralNetwork) Verify(samples []*TrainingSample) bool {
	fitness := neuralNet.Fitness(samples)
	return fitness >= FITNESS_THRESHOLD
}

func (neuralNet *NeuralNetwork) Run() {

	// get list of unique nodes in network
	nodes := neuralNet.uniqueNodeMap()

	// call Run() on each node
	for node, _ := range nodes {
		if node.isInvisible() == false {
			node.Run()
		}
	}

}

func (neuralNet *NeuralNetwork) Shutdown() {

	// get list of unique nodes in network
	nodes := neuralNet.uniqueNodeMap()

	// call Run() on each node
	for node, _ := range nodes {
		node.Shutdown()
	}

}

func (neuralNet *NeuralNetwork) MarshalJSON() ([]byte, error) {

	// get list of unique nodes in network
	nodes := neuralNet.uniqueNodeMap()

	nodeSlice := make([]*Node, 0)
	for node, _ := range nodes {
		nodeSlice = append(nodeSlice, node)
	}
	return json.Marshal(nodeSlice)
}

func (neuralNet *NeuralNetwork) Neurons() []*Node {

	neurons := make([]*Node, 0)
	nodes := neuralNet.uniqueNodeMap()
	for _, node := range nodes {
		if node.processor.HasBias() { // <-- mild hack
			neurons = append(neurons, node)
		}
	}
	return neurons
}

func (neuralNet *NeuralNetwork) SetSensors(sensors []*Node) {
	neuralNet.sensors = sensors
}

func (neuralNet *NeuralNetwork) SetActuators(actuators []*Node) {
	neuralNet.actuators = actuators
}

func (neuralNet *NeuralNetwork) uniqueNodeMap() NodeMap {
	uniqueNodeMap := make(NodeMap)
	for _, sensor := range neuralNet.sensors {
		neuralNet.addUniqueNodeRecursive(sensor, uniqueNodeMap)
	}
	return uniqueNodeMap
}

func (neuralNet *NeuralNetwork) addUniqueNodeRecursive(node *Node, uniqueNodeMap NodeMap) {
	if _, ok := uniqueNodeMap[node]; !ok {
		if node.isInvisible() == false {
			uniqueNodeMap[node] = node
		}
	}
	for _, connection := range node.outbound {
		neuralNet.addUniqueNodeRecursive(connection.other, uniqueNodeMap)
	}
}

func (neuralNet *NeuralNetwork) NumLayers() int {
	return len(neuralNet.NodesByLayer())
}

// Get all the nodes of this network grouped by layer.  The slice
// that is returned has all the nodes for that layer index, eg,
// [0] -> [sensor1, sensor2]
// [1] -> [neuron1, neuron2]
// [2] -> [actuator]
func (neuralNet *NeuralNetwork) NodesByLayer() [][]*Node {

	nodesByLayer := [][]*Node{neuralNet.sensors}

	thisLayerNodes := neuralNet.sensors

	for {
		nextLayerNodes := neuralNet.nextLayerNodes(thisLayerNodes)
		if len(nextLayerNodes) == 0 {
			break
		}
		nodesByLayer = append(nodesByLayer, nextLayerNodes)
		thisLayerNodes = nextLayerNodes
	}

	return nodesByLayer
}

func (neuralNet *NeuralNetwork) NodesInLayer(layerIndex int) []*Node {
	nodesByLayer := neuralNet.NodesByLayer()
	return nodesByLayer[layerIndex]
}

func (neuralNet *NeuralNetwork) nextLayerNodes(layerNodes []*Node) []*Node {

	uniqueNodeMap := make(NodeMap)
	for _, node := range layerNodes {
		for _, connection := range node.outbound {
			outboundNode := connection.other
			if _, ok := uniqueNodeMap[outboundNode]; !ok {
				if outboundNode.isInvisible() == false {
					uniqueNodeMap[outboundNode] = outboundNode
				}
			}

		}
	}

	nextLayerNodes := make([]*Node, len(uniqueNodeMap))
	i := 0
	for key, _ := range uniqueNodeMap {
		nextLayerNodes[i] = key
		i++
	}

	return nextLayerNodes

}

func (neuralNet *NeuralNetwork) Copy() *NeuralNetwork {

	// the source neural network provides a "scaffold" for building the
	// target network.  these provide the mapping between nodes and channels.
	nodeScaffold := make(map[*Node]*Node)
	channelScaffold := make(map[VectorChannel]VectorChannel)

	scaffold := &copyScaffold{nodeScaffold: nodeScaffold, channelScaffold: channelScaffold}

	sensorsCopy := make([]*Node, 0)
	actuatorsCopy := make([]*Node, 0)
	neuralNetCopy := &NeuralNetwork{sensors: sensorsCopy, actuators: actuatorsCopy}

	for _, sensor := range neuralNet.sensors {
		sensorCopy := new(Node)
		sensorCopy.processor = sensor.processor.copy()
		nodeScaffold[sensor] = sensorCopy
		sensorCopy.Name = sensor.Name
		neuralNetCopy.sensors = append(neuralNetCopy.sensors, sensorCopy)
	}

	for _, actuator := range neuralNet.actuators {
		actuatorCopy := new(Node)
		actuatorCopy.processor = actuator.processor.copy()
		nodeScaffold[actuator] = actuatorCopy
		actuatorCopy.Name = actuator.Name
		neuralNetCopy.actuators = append(neuralNetCopy.actuators, actuatorCopy)
	}

	for _, sensor := range neuralNet.sensors {
		sensorCopy := nodeScaffold[sensor]
		recreateOutboundConnectionsRecursive(sensor, sensorCopy, scaffold)
	}

	for _, actuator := range neuralNet.actuators {
		actuatorCopy := nodeScaffold[actuator]
		recreateInboundConnectionsRecursive(actuator, actuatorCopy, scaffold)
	}

	return neuralNetCopy

}

func recreateInboundConnectionsRecursive(nodeOriginal *Node, nodeCopy *Node, scaffold *copyScaffold) {

	nodeScaffold := scaffold.nodeScaffold
	channelScaffold := scaffold.channelScaffold

	for _, inboundConnection := range nodeOriginal.inboundConnections() {

		cxnTargetOriginal := inboundConnection.other
		cxnTargetCopy := createOrReuseConnectionTargetCopy(cxnTargetOriginal, nodeScaffold)

		newCxn := &Connection{}
		newCxn.other = cxnTargetCopy

		channelCopy := createChannelCopy(inboundConnection.channel, channelScaffold)
		newCxn.channel = channelCopy

		if inboundConnection.weights != nil && len(inboundConnection.weights) > 0 {
			weightsCopy := make([]float64, len(inboundConnection.weights))
			copy(weightsCopy, inboundConnection.weights)
			newCxn.weights = weightsCopy
		}

		nodeCopy.appendInboundConnection(newCxn)

		if len(cxnTargetOriginal.inboundConnections()) > 0 {
			recreateInboundConnectionsRecursive(cxnTargetOriginal, cxnTargetCopy, scaffold)
		}
	}
}

func recreateOutboundConnectionsRecursive(nodeOriginal *Node, nodeCopy *Node, scaffold *copyScaffold) {

	nodeScaffold := scaffold.nodeScaffold
	channelScaffold := scaffold.channelScaffold

	for _, outboundConnection := range nodeOriginal.outboundConnections() {

		cxnTargetOriginal := outboundConnection.other

		cxnTargetCopy := createOrReuseConnectionTargetCopy(cxnTargetOriginal, nodeScaffold)

		if !nodeCopy.hasOutboundConnectionTo(cxnTargetCopy) {

			newCxn := &Connection{}
			newCxn.other = cxnTargetCopy

			channelCopy := createChannelCopy(outboundConnection.channel, channelScaffold)
			newCxn.channel = channelCopy

			nodeCopy.appendOutboundConnection(newCxn)

		}

		if len(cxnTargetOriginal.outboundConnections()) > 0 {
			recreateOutboundConnectionsRecursive(cxnTargetOriginal, cxnTargetCopy, scaffold)
		}

	}
}

func createChannelCopy(channelOriginal VectorChannel, channelScaffold map[VectorChannel]VectorChannel) VectorChannel {

	var channelCopy VectorChannel
	if channelCopyTemp, ok := channelScaffold[channelOriginal]; ok {
		channelCopy = channelCopyTemp
	} else {
		channelCopy = make(VectorChannel)
		channelScaffold[channelOriginal] = channelCopy
	}
	return channelCopy

}

func createOrReuseConnectionTargetCopy(cxnTargetOriginal *Node, nodeScaffold map[*Node]*Node) *Node {

	var cxnTargetCopy *Node
	if cxnTargetCopyTemp, ok := nodeScaffold[cxnTargetOriginal]; ok { // TODO: hack
		cxnTargetCopy = cxnTargetCopyTemp
	} else {

		// the connection target does not exist in nodeScaffold, create it
		node := &Node{}
		node.Name = cxnTargetOriginal.Name
		cxnTargetCopy = node
		cxnTargetCopy.processor = cxnTargetOriginal.processor.copy()

		nodeScaffold[cxnTargetOriginal] = cxnTargetCopy

	}

	return cxnTargetCopy

}
