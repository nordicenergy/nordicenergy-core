package genesis

import "testing"

func TestLocalTestAccounts(t *testing.T) {
	for name, accounts := range map[string][]DeployAccount{
		"nordicenergyV0":      LocalnordicenergyAccounts,
		"nordicenergyV1":      LocalnordicenergyAccountsV1,
		"nordicenergyV2":      LocalnordicenergyAccountsV2,
		"FoundationalV0": LocalFnAccounts,
		"FoundationalV1": LocalFnAccountsV1,
		"FoundationalV2": LocalFnAccountsV2,
	} {
		t.Run(name, func(t *testing.T) { testDeployAccounts(t, accounts) })
	}
}
