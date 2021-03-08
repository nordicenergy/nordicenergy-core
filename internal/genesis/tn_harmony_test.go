package genesis

import "testing"

func TestTNnordicenergyAccounts(t *testing.T) {
	testDeployAccounts(t, TNnordicenergyAccounts)
}

func TestTNFoundationalAccounts(t *testing.T) {
	testDeployAccounts(t, TNFoundationalAccounts)
}
