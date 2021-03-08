package genesis

import "testing"

func TestTNNordicEnergyAccounts(t *testing.T) {
	testDeployAccounts(t, TNNordicEnergyAccounts)
}

func TestTNFoundationalAccounts(t *testing.T) {
	testDeployAccounts(t, TNFoundationalAccounts)
}
