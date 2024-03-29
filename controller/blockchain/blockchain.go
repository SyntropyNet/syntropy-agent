package blockchain

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/state"
	"github.com/cosmos/go-bip39"
	"github.com/decred/base58"

	"github.com/SyntropyNet/syntropy-agent/controller"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"

	ipfsApi "github.com/ipfs/go-ipfs-api"
)

const (
	pkgName        = "Blockchain Controller. "
	ipfsUrl        = "https://ipfs.io/ipfs/"
	mnemonicPath   = env.AgentConfigDir + "/mnemonic"
	addressPath    = env.AgentConfigDir + "/address"
	reconnectDelay = 10000 // 10 seconds (in milliseconds)
	waitForMsg     = time.Second
)

const (
	// State machine constants
	stopped = iota
	connecting
	running
)

var ErrNotRunning = errors.New("substrate api is not running")

// Blockchain controller. To be implemented in future
type BlockchainController struct {
	sync.Mutex
	state.StateMachine
	substrateApi *gsrpc.SubstrateAPI
	keyringPair  signature.KeyringPair
	ipfsShell    *ipfsApi.Shell
	genesisHash  types.Hash
	metadata     *types.Metadata
	comodityKey  types.StorageKey
	systemKey    types.StorageKey

	url           string
	lastCommodity []byte
}

type BlockchainMsg struct {
	Url string `json:"url"`
	Cid string `json:"cid"`
}

type Commodity struct {
	ID      types.Hash
	Payload []byte
}

type CommodityInfo struct {
	Info []byte
}

func getIpfsPayload(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func getMnemonic() (string, error) {
	content, err := os.ReadFile(mnemonicPath)
	if err == nil {
		return string(content), nil
	}

	// Mnemonic cache does not exist - create new
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		// cannot reuse wallet in future
		logger.Error().Println(pkgName, "Could not generate new entropy", err)
		return "", err
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		// cannot reuse wallet in future
		logger.Error().Println(pkgName, "Could not generate new mnemonic", err)
		return "", err
	}

	err = os.WriteFile(mnemonicPath, []byte(mnemonic), 0600)
	if err != nil {
		// cannot reuse wallet in future
		logger.Error().Println(pkgName, "Mnemonic cache", err)
		return "", err
	}

	return mnemonic, nil
}

func New() (controller.Controller, error) {
	bc := BlockchainController{
		url: config.GetCloudURL(),
	}

	mnemonic, err := getMnemonic()
	if err != nil {
		return nil, err
	}

	// TODO: what is 42 here ???
	bc.keyringPair, err = signature.KeyringPairFromSecret(mnemonic, 42)
	if err != nil {
		logger.Error().Println(pkgName, "Keyring from secret", err)
		return nil, err
	}

	// Always update address file with latest content.
	// NOTE: other scripts need this address to put tokens there
	err = os.WriteFile(addressPath, []byte(bc.keyringPair.Address), 0600)
	if err != nil {
		// Cannot work with blockchain, since other scripts cannot put tokens to wallet
		logger.Error().Println(pkgName, "Wallet address cache", err)
		return nil, err
	}

	return &bc, nil
}

func (bc *BlockchainController) Open() error {
	return bc.connect()
}

func (bc *BlockchainController) connect() error {
	var err error
	logger.Debug().Println(pkgName, "Connecting to Substrate API...")
	bc.SetState(connecting)
	for {
		bc.substrateApi, err = gsrpc.NewSubstrateAPI(bc.url)
		if err != nil {
			logger.Error().Println(pkgName, "ConnectionError", err)
			// Add some randomised sleep, so if controller was down
			// the reconnecting agents could DDOS the controller
			delay := time.Duration(rand.Int31n(reconnectDelay)) * time.Millisecond
			logger.Warning().Println(pkgName, "Reconnecting in ", delay)
			time.Sleep(delay)
			continue
		}
		logger.Info().Println(pkgName, "Connected to Substrate API")
		break
	}

	// Get and store values that do not change
	bc.genesisHash, err = bc.substrateApi.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return err
	}

	bc.metadata, err = bc.substrateApi.RPC.State.GetMetadataLatest()
	if err != nil {
		logger.Error().Println(pkgName, "metadata latest", err)
		return err
	}

	bc.systemKey, err = types.CreateStorageKey(bc.metadata, "System", "Account", bc.keyringPair.PublicKey, nil)
	if err != nil {
		logger.Error().Println(pkgName, "storage key", err)
		return err
	}

	bc.comodityKey, err = types.CreateStorageKey(bc.metadata, "Commodity", "CommoditiesForAccount", bc.keyringPair.PublicKey, nil)
	if err != nil {
		logger.Error().Println(pkgName, "comodity key", err)
		return err
	}

	// Init IPFS
	bc.ipfsShell = ipfsApi.NewShell(config.GetIpfsUrl())

	// Mark running (do I need this state in this controller ?)
	bc.SetState(running)

	return nil
}

func (bc *BlockchainController) Recv() ([]byte, error) {
	if bc.GetState() == stopped {
		return nil, ErrNotRunning
	}

	var res []Commodity
	var msg BlockchainMsg

	// TODO: change this to some listener (I bet the substrate package has one)
	for {
		ok, err := bc.substrateApi.RPC.State.GetStorageLatest(bc.comodityKey, &res)
		if err != nil {
			logger.Error().Println(pkgName, "get comodity", err)
			continue
		}

		if !ok || len(res) == 0 {
			time.Sleep(waitForMsg)
			continue
		}

		if bytes.Equal(bc.lastCommodity, res[len(res)-1].Payload) {
			time.Sleep(waitForMsg)
			continue
		}

		bc.lastCommodity = res[len(res)-1].Payload
		err = json.Unmarshal(bc.lastCommodity, &msg)

		if err != nil {
			logger.Error().Println(pkgName, "substrate comodity unmarshal", err)
			continue
		}
		// Extract the payload from IPFS
		// TODO: signature and encryption ?
		data, err := getIpfsPayload(msg.Url)
		if err != nil {
			logger.Error().Println(pkgName, "IPFS payload", err)
			continue
		}

		return data, nil
	}
}

func (bc *BlockchainController) Write(b []byte) (n int, err error) {
	if controllerState := bc.GetState(); controllerState != running {
		logger.Warning().Println(pkgName, "Controller is not running. Current state: ", controllerState)
		return 0, ErrNotRunning
	}

	// TODO: signature and encryption ?
	cid, err := bc.ipfsShell.Add(bytes.NewReader(b))
	if err != nil {
		logger.Error().Println(pkgName, "IPFS file hash", err)
		return 0, err
	}
	ipfsUrl := ipfsUrl + cid

	msg, err := json.Marshal(BlockchainMsg{
		Url: ipfsUrl,
		Cid: cid,
	})
	if err != nil {
		logger.Error().Println(pkgName, "JSON marshal error", err)
		return 0, err
	}

	var accountInfo types.AccountInfo
	ok, err := bc.substrateApi.RPC.State.GetStorageLatest(bc.systemKey, &accountInfo)
	if err != nil || !ok {
		logger.Error().Println(pkgName, "account info", err)
		return 0, err
	}

	rv, err := bc.substrateApi.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		logger.Error().Println(pkgName, "runtime version", err)
		return 0, err
	}

	c, err := types.NewCall(bc.metadata, "Commodity.mint", types.NewAccountID(base58.Decode(config.GetOwnerAddress())[1:33]), msg)
	if err != nil {
		logger.Error().Println(pkgName, "commodity mint", err)
		return 0, err
	}

	ext := types.NewExtrinsic(c)
	err = ext.Sign(bc.keyringPair, types.SignatureOptions{
		BlockHash:          bc.genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		GenesisHash:        bc.genesisHash,
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	})
	if err != nil {
		logger.Error().Println(pkgName, "signing", err)
		return 0, err
	}

	_, err = bc.substrateApi.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		logger.Error().Println(pkgName, "submit", err)
		return 0, err
	}

	return len(b), nil
}

func (bc *BlockchainController) Close() error {
	if bc.GetState() == stopped {
		// cannot close already closed connection
		return ErrNotRunning
	}
	bc.SetState(stopped)

	// Maybe notify blockchain about closing?
	return nil
}
