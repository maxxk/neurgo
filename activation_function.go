package neurgo

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
)

type ActivationFunction func(float64) float64

type EncodableActivation struct {
	Name               string
	ActivationFunction ActivationFunction
}

func (activation *EncodableActivation) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		struct {
			Name string
		}{
			Name: activation.Name,
		})
}

func (activation *EncodableActivation) UnmarshalJSON(bytes []byte) error {

	rawMap := make(map[string]interface{})
	err := json.Unmarshal(bytes, &rawMap)
	if err != nil {
		return err
	}

	// TODO: isn't there an easier / less brittle way to do this??
	var ok bool
	if activation.Name, ok = rawMap["Name"].(string); !ok {
		log.Panicf("Could not unmarshal %v into EncodableActivation", rawMap)
	}

	switch activation.Name {
	case "sigmoid":
		activation.ActivationFunction = Sigmoid
	case "tanh":
		activation.ActivationFunction = math.Tanh
	case "identity":
		activation.ActivationFunction = Identity
	case "relu":
		activation.ActivationFunction = ReLU
	case "logistic":
		activation.ActivationFunction = Logistic
	case "abs":
		activation.ActivationFunction = math.Abs
	case "gaussian":
		activation.ActivationFunction = Gaussian
	default:
		log.Panicf("Unknown activation function: %v", activation.Name)
	}

	return nil
}

func (activation *EncodableActivation) String() string {
	return fmt.Sprintf("%v (%v)", activation.Name, activation.ActivationFunction)
}

func Sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Pow(math.E, -1.0*x))
}

func EncodableSigmoid() *EncodableActivation {
	return &EncodableActivation{
		Name:               "sigmoid",
		ActivationFunction: Sigmoid,
	}
}

func Identity(x float64) float64 {
	return x
}

func EncodableIdentity() *EncodableActivation {
	return &EncodableActivation{
		Name:               "identity",
		ActivationFunction: Identity,
	}
}

func EncodableTanh() *EncodableActivation {
	return &EncodableActivation{
		Name:               "tanh",
		ActivationFunction: math.Tanh,
	}
}

func ReLU(x float64) float64 {
	return math.Max(x, 0)
}

func EncodableReLU() *EncodableActivation {
	return &EncodableActivation {
		Name:            "relu",
		ActivationFunction: ReLU,
	}
}

func Logistic(x float64) float64 {
	return float64(1.0) / (1.0 + math.Exp(-x))
}

func EncodableLogistic() *EncodableActivation {
	return &EncodableActivation {
		Name:        "logistic",
		ActivationFunction: Logistic,
	}
}

func EncodableAbs() *EncodableActivation {
	return &EncodableActivation {
		Name:       "abs",
		ActivationFunction: math.Abs,
	}
}

func Gaussian(x float64) float64 {
	return math.Exp(-16*x*x)
}

func EncodableGaussian() *EncodableActivation {
	return &EncodableActivation {
		Name:      "gaussian",
		ActivationFunction: Gaussian,
	}
}

func AllEncodableActivations() []*EncodableActivation {
	return []*EncodableActivation{EncodableSigmoid(), EncodableTanh(), EncodableReLU(), EncodableLogistic(), EncodableIdentity(), EncodableGaussian(), EncodableAbs()}
}

func RandomEncodableActivation() *EncodableActivation {
	allActivations := AllEncodableActivations()
	randIndex := RandomIntInRange(0, len(allActivations))
	return allActivations[randIndex]
}
