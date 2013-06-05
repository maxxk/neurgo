
package neurgo

import (
	"log"
	// "github.com/proxypoke/vector"
)

type activationFunction func(float64) float64

type Neuron struct {
	Bias               float64
	ActivationFunction activationFunction
	Node
}

	
func (neuron *Neuron) computeOutput(weightedInputs []*weightedInput) float64 {

    /*
    reduce_function = fn({inputs, weights}, acc) ->
                          dot_product(inputs, weights) + acc
                      end
    output = Enum.reduce weighted_inputs, 0, reduce_function
    output = output + bias
    activation_function.(output)
    */
	for i, weightedInput := range weightedInputs {
		log.Printf("i: %v, weightedInput: %v", i, weightedInput)

	}

	return 0
	

}
