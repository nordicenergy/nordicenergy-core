package rpc

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/nordicenergy/nordicenergy-core/ngy"
	nodeconfig "github.com/nordicenergy/nordicenergy-core/internal/configs/node"
	"github.com/nordicenergy/nordicenergy-core/internal/utils"
	eth "github.com/nordicenergy/nordicenergy-core/rpc/eth"
	v1 "github.com/nordicenergy/nordicenergy-core/rpc/v1"
	v2 "github.com/nordicenergy/nordicenergy-core/rpc/v2"
)

// Version enum
const (
	V1 Version = iota
	V2
	Eth
	Debug
)

const (
	// APIVersion used for DApp's, bumped after RPC refactor (7/2020)
	APIVersion = "1.1"
	// CallTimeout is the timeout given to all contract calls
	CallTimeout = 5 * time.Second
	// LogTag is the tag found in the log for all RPC logs
	LogTag = "[RPC]"
	// HTTPPortOffset ..
	HTTPPortOffset = 500
	// WSPortOffset ..
	WSPortOffset = 800

	netNamespace   = "net"
	netV1Namespace = "netv1"
	netV2Namespace = "netv2"
	web3Namespace  = "web3"
)

var (
	// HTTPModules ..
	HTTPModules = []string{"ngy", "ngyv2", "eth", "debug", netNamespace, netV1Namespace, netV2Namespace, web3Namespace, "explorer"}
	// WSModules ..
	WSModules = []string{"ngy", "ngyv2", "eth", "debug", netNamespace, netV1Namespace, netV2Namespace, web3Namespace, "web3"}

	httpListener     net.Listener
	httpHandler      *rpc.Server
	wsListener       net.Listener
	wsHandler        *rpc.Server
	httpEndpoint     = ""
	wsEndpoint       = ""
	httpVirtualHosts = []string{"*"}
	httpTimeouts     = rpc.DefaultHTTPTimeouts
	httpOrigins      = []string{"*"}
	wsOrigins        = []string{"*"}
)

// Version of the RPC
type Version int

// Namespace of the RPC version
func (n Version) Namespace() string {
	return HTTPModules[n]
}

// StartServers starts the http & ws servers
func StartServers(ngy *ngy.nordicenergy, apis []rpc.API, config nodeconfig.RPCServerConfig) error {
	apis = append(apis, getAPIs(ngy, config.DebugEnabled)...)

	if config.HTTPEnabled {
		httpEndpoint = fmt.Sprintf("%v:%v", config.HTTPIp, config.HTTPPort)
		if err := startHTTP(apis); err != nil {
			return err
		}
	}

	if config.WSEnabled {
		wsEndpoint = fmt.Sprintf("%v:%v", config.WSIp, config.WSPort)
		if err := startWS(apis); err != nil {
			return err
		}
	}

	return nil
}

// StopServers stops the http & ws servers
func StopServers() error {
	if httpListener != nil {
		if err := httpListener.Close(); err != nil {
			return err
		}
		httpListener = nil
		utils.Logger().Info().
			Str("url", fmt.Sprintf("http://%s", httpEndpoint)).
			Msg("HTTP endpoint closed")
	}
	if httpHandler != nil {
		httpHandler.Stop()
		httpHandler = nil
	}
	if wsListener != nil {
		if err := wsListener.Close(); err != nil {
			return err
		}
		wsListener = nil
		utils.Logger().Info().
			Str("url", fmt.Sprintf("http://%s", wsEndpoint)).
			Msg("WS endpoint closed")
	}
	if wsHandler != nil {
		wsHandler.Stop()
		wsHandler = nil
	}
	return nil
}

// getAPIs returns all the API methods for the RPC interface
func getAPIs(ngy *ngy.nordicenergy, debugEnable bool) []rpc.API {
	publicAPIs := []rpc.API{
		// Public methods
		NewPublicnordicenergyAPI(ngy, V1),
		NewPublicnordicenergyAPI(ngy, V2),
		NewPublicnordicenergyAPI(ngy, Eth),
		NewPublicBlockchainAPI(ngy, V1),
		NewPublicBlockchainAPI(ngy, V2),
		NewPublicBlockchainAPI(ngy, Eth),
		NewPublicContractAPI(ngy, V1),
		NewPublicContractAPI(ngy, V2),
		NewPublicContractAPI(ngy, Eth),
		NewPublicTransactionAPI(ngy, V1),
		NewPublicTransactionAPI(ngy, V2),
		NewPublicTransactionAPI(ngy, Eth),
		NewPublicPoolAPI(ngy, V1),
		NewPublicPoolAPI(ngy, V2),
		NewPublicPoolAPI(ngy, Eth),
		NewPublicStakingAPI(ngy, V1),
		NewPublicStakingAPI(ngy, V2),
		NewPublicTracerAPI(ngy, Debug),
		// Legacy methods (subject to removal)
		v1.NewPublicLegacyAPI(ngy, "ngy"),
		eth.NewPublicEthService(ngy, "eth"),
		v2.NewPublicLegacyAPI(ngy, "ngyv2"),
	}

	privateAPIs := []rpc.API{
		NewPrivateDebugAPI(ngy, V1),
		NewPrivateDebugAPI(ngy, V2),
	}

	if debugEnable {
		return append(publicAPIs, privateAPIs...)
	}
	return publicAPIs
}

func startHTTP(apis []rpc.API) (err error) {
	httpListener, httpHandler, err = rpc.StartHTTPEndpoint(
		httpEndpoint, apis, HTTPModules, httpOrigins, httpVirtualHosts, httpTimeouts,
	)
	if err != nil {
		return err
	}

	utils.Logger().Info().
		Str("url", fmt.Sprintf("http://%s", httpEndpoint)).
		Str("cors", strings.Join(httpOrigins, ",")).
		Str("vhosts", strings.Join(httpVirtualHosts, ",")).
		Msg("HTTP endpoint opened")
	fmt.Printf("Started RPC server at: %v\n", httpEndpoint)
	return nil
}

func startWS(apis []rpc.API) (err error) {
	wsListener, wsHandler, err = rpc.StartWSEndpoint(wsEndpoint, apis, WSModules, wsOrigins, true)
	if err != nil {
		return err
	}

	utils.Logger().Info().
		Str("url", fmt.Sprintf("ws://%s", wsListener.Addr())).
		Msg("WebSocket endpoint opened")
	fmt.Printf("Started WS server at: %v\n", wsEndpoint)
	return nil
}
