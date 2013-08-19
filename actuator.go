package neurgo

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

type ActuatorFunction func(outputs []float64)

type Actuator struct {
	NodeId           *NodeId
	Inbound          []*InboundConnection
	Closing          chan chan bool
	DataChan         chan *DataMessage
	VectorLength     int
	ActuatorFunction ActuatorFunction
	wg               *sync.WaitGroup
	Cortex           *Cortex
}

func (actuator *Actuator) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		struct {
			NodeId       *NodeId
			VectorLength int
			Inbound      []*InboundConnection
		}{
			NodeId:       actuator.NodeId,
			VectorLength: actuator.VectorLength,
			Inbound:      actuator.Inbound,
		})
}

func (actuator *Actuator) String() string {
	return JsonString(actuator)
}

func (actuator *Actuator) Run() {

	defer actuator.wg.Done()

	actuator.checkRunnable()

	weightedInputs := createEmptyWeightedInputs(actuator.Inbound)

	closed := false

	for {

		select {
		case responseChan := <-actuator.Closing:
			closed = true
			responseChan <- true
			break // TODO: do we need this for anything??
		case dataMessage := <-actuator.DataChan:
			recordInput(weightedInputs, dataMessage)
		}

		if closed {
			actuator.Closing = nil
			actuator.DataChan = nil
			break
		}

		if receiveBarrierSatisfied(weightedInputs) {

			scalarOutput := actuator.computeScalarOutput(weightedInputs)

			actuator.ActuatorFunction(scalarOutput)

			weightedInputs = createEmptyWeightedInputs(actuator.Inbound)

		}

	}

}

func (actuator *Actuator) Shutdown() {

	closingResponse := make(chan bool)
	actuator.Closing <- closingResponse
	response := <-closingResponse
	if response != true {
		log.Panicf("Got unexpected response on closing channel")
	}

	actuator.wg.Wait()
	actuator.wg = nil
}

func (actuator *Actuator) computeScalarOutput(weightedInputs []*weightedInput) []float64 {

	outputs := make([]float64, 0)
	for _, weightedInput := range weightedInputs {
		inputs := weightedInput.inputs
		actuator.validateInputs(inputs)
		inputValue := inputs[0]
		outputs = append(outputs, inputValue)
	}

	return outputs

}

func (actuator *Actuator) validateInputs(inputs []float64) {
	if len(inputs) != 1 {
		t := "%T got invalid input vector: %v"
		message := fmt.Sprintf(t, actuator, inputs)
		panic(message)
	}
}

func (actuator *Actuator) checkRunnable() {
	if actuator.NodeId == nil {
		msg := fmt.Sprintf("not expecting actuator.NodeId to be nil")
		panic(msg)
	}

	if actuator.Closing == nil {
		msg := fmt.Sprintf("not expecting actuator.Closing to be nil")
		panic(msg)
	}

	if actuator.DataChan == nil {
		msg := fmt.Sprintf("not expecting actuator.DataChan to be nil")
		panic(msg)
	}

	if actuator.ActuatorFunction == nil {
		msg := fmt.Sprintf("not expecting actuator.ActuatorFunction to be nil")
		panic(msg)
	}

	if len(actuator.Inbound) != actuator.VectorLength {
		msg := fmt.Sprintf("# of inbound (%d) != VectorLength (%d)",
			len(actuator.Inbound),
			actuator.VectorLength)
		panic(msg)
	}

}

func (actuator *Actuator) Init(reInit bool) {
	if reInit == true {
		actuator.Closing = make(chan chan bool)
	} else if actuator.Closing == nil {
		actuator.Closing = make(chan chan bool)
	}

	if reInit == true {
		actuator.DataChan = make(chan *DataMessage, len(actuator.Inbound))
	} else if actuator.DataChan == nil {
		actuator.DataChan = make(chan *DataMessage, len(actuator.Inbound))
	}

	if reInit == true {
		actuatorFunc := func(outputs []float64) {
			log.Panicf("defualt actuator function called - do nothing")
		}
		actuator.ActuatorFunction = actuatorFunc
	} else if actuator.ActuatorFunction == nil {
		// if there is no ActuatorFunction, create a default
		// function which does nothing
		actuatorFunc := func(outputs []float64) {
			log.Panicf("defualt actuator function called - do nothing")
		}
		actuator.ActuatorFunction = actuatorFunc
	}

	if reInit == true {
		actuator.wg = &sync.WaitGroup{}
		actuator.wg.Add(1)
	} else if actuator.wg == nil {
		actuator.wg = &sync.WaitGroup{}
		actuator.wg.Add(1)
	}

}

func (actuator *Actuator) ConnectInbound(connectable InboundConnectable) {
	if actuator.Inbound == nil {
		actuator.Inbound = make([]*InboundConnection, 0)
	}
	connection := &InboundConnection{
		NodeId:  connectable.nodeId(),
		Weights: nil,
	}
	actuator.Inbound = append(actuator.Inbound, connection)
}

func (actuator *Actuator) inbound() []*InboundConnection {
	return actuator.Inbound
}

func (actuator *Actuator) setInbound(newInbound []*InboundConnection) {
	actuator.Inbound = newInbound
}

func (actuator *Actuator) dataChan() chan *DataMessage {
	return actuator.DataChan
}

func (actuator *Actuator) nodeId() *NodeId {
	return actuator.NodeId
}

func (actuator *Actuator) CanAddInboundConnection() bool {
	return len(actuator.Inbound) < actuator.VectorLength
}
