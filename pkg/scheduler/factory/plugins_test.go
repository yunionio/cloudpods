package factory

import (
	"testing"
)

func TestAlgorithmNameValidation(t *testing.T) {
	namesShouldValidate := []string{
		"1SomeAlgo1rithm",
		"someAlgor-ithm1",
	}
	namesShouldNotValidate := []string{
		"-SomeAlgorithm",
		"SomeAlgorithm-",
		"Some,Alg:orithm",
	}
	for _, name := range namesShouldValidate {
		if !validName.MatchString(name) {
			t.Errorf("%v should be a valid algorithm name but is not valid.", name)
		}
	}
	for _, name := range namesShouldNotValidate {
		if validName.MatchString(name) {
			t.Errorf("%v should be an invalid algorithm name but is valid", name)
		}
	}
}
