package neurgo

import (
	"encoding/json"
	"fmt"
	"github.com/couchbaselabs/go.assert"
	"log"
	"testing"
	"time"
)

func TestSensorJsonMarshal(t *testing.T) {

	fakeNodeId := NewNeuronId("fake-node", 0.25)
	dataChan := make(chan *DataMessage, 1)
	outboundConnection := &OutboundConnection{
		NodeId:   fakeNodeId,
		DataChan: dataChan,
	}

	sensorFunc := func(syncCounter int) []float64 {
		return []float64{0}
	}

	sensorNodeId := NewSensorId("sensor", 0.0)
	sensor := &Sensor{
		NodeId:         sensorNodeId,
		VectorLength:   2,
		SensorFunction: sensorFunc,
		Outbound:       []*OutboundConnection{outboundConnection},
	}

	json, err := json.Marshal(sensor)
	if err != nil {
		log.Fatal(err)
	}
	assert.True(t, err == nil)
	jsonString := fmt.Sprintf("%s", json)
	log.Printf("jsonString: %v", jsonString)
}

func TestSensorRun(t *testing.T) {

	fakeNodeId := NewNeuronId("fake-node", 0.25)
	dataChan := make(chan *DataMessage, 1)
	outboundConnection := &OutboundConnection{
		NodeId:   fakeNodeId,
		DataChan: dataChan,
	}

	sensorNodeId := NewSensorId("sensor", 0.0)

	numTimesFuncCalled := 0
	sensorFunc := func(syncCounter int) []float64 {
		numTimesFuncCalled += 1
		return []float64{float64(syncCounter)}
	}

	sensor := &Sensor{
		NodeId:         sensorNodeId,
		VectorLength:   2,
		SensorFunction: sensorFunc,
		Outbound:       []*OutboundConnection{outboundConnection},
	}
	sensor.Init()
	go sensor.Run()

	// send it a sync message
	sensor.SyncChan <- true

	// get value from dataChan and make sure its expected value
	select {
	case dataMessage := <-dataChan:
		output := dataMessage.Inputs
		assert.True(t, vectorEqualsWithMaxDelta(output, []float64{0}, 0.001))
	case <-time.After(time.Second):
		assert.Errorf(t, "Got unexpected timeout")
	}
	assert.Equals(t, numTimesFuncCalled, 1)

	// send it a sync message
	sensor.SyncChan <- true

	// get value from dataChan and make sure its expected value
	select {
	case dataMessage := <-dataChan:
		output := dataMessage.Inputs
		assert.True(t, vectorEqualsWithMaxDelta(output, []float64{1}, 0.001))
	case <-time.After(time.Second):
		assert.Errorf(t, "Got unexpected timeout")
	}
	assert.Equals(t, numTimesFuncCalled, 2)

	sensor.Shutdown()

}
