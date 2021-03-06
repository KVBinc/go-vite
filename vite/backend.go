package vite

import (
	ledgerHandler "github.com/vitelabs/go-vite/ledger/handler"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/protocols"
	"github.com/vitelabs/go-vite/wallet"

	"github.com/vitelabs/go-vite/ledger/handler_interface"
	protoInterface "github.com/vitelabs/go-vite/protocols/interfaces"

	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/miner"
	"github.com/vitelabs/go-vite/signer"
	"time"
)

type Vite struct {
	config        *config.Config
	ledger        *ledgerHandler.Manager
	p2p           *p2p.Server
	pm            *protocols.ProtocolManager
	walletManager *wallet.Manager
	signer        *signer.Master
	verifier      consensus.Verifier
	miner         *miner.Miner
}

//
//var (
//	defaultP2pConfig = &p2p.Config{}
//	DefaultConfig    = &Config{
//		DataDir:   common.DefaultDataDir(),
//		P2pConfig: defaultP2pConfig,
//		Miner:     false,
//		Coinbase:  "",
//	}
//)
//
//type Config struct {
//	DataDir       string
//	P2pConfig     *p2p.Config
//	Miner         bool
//	Coinbase      string
//	MinerInterval int
//}

func New(cfg *config.Config) (*Vite, error) {
	log := log15.New("module", "vite/backend")
	vite := &Vite{config: cfg}

	vite.ledger = ledgerHandler.NewManager(vite, cfg.DataDir)

	vite.walletManager = wallet.NewManagerAndInit(cfg.DataDir)
	vite.signer = signer.NewMaster(vite)
	vite.signer.InitAndStartLoop()

	vite.pm = protocols.NewProtocolManager(vite)

	var initP2pErr error
	vite.p2p, initP2pErr = p2p.NewServer(&cfg.P2P, vite.pm.HandlePeer)
	if initP2pErr != nil {
		log.Crit(initP2pErr.Error())
	}

	genesisTime := time.Unix(int64(ledger.GetSnapshotGenesisBlock().Timestamp), 0)
	committee := consensus.NewCommittee(genesisTime, int32(cfg.MinerInterval), int32(len(consensus.DefaultMembers)))
	vite.verifier = committee

	if cfg.Miner.Miner && cfg.Miner.Coinbase != "" {
		log.Info("Vite backend new: Start miner.")
		coinbase, _ := types.HexToAddress(cfg.Miner.Coinbase)
		vite.miner = miner.NewMiner(vite.ledger.Sc(), vite.ledger.RegisterFirstSyncDown, coinbase, committee)
		pwd := "123"
		vite.walletManager.KeystoreManager.Unlock(coinbase, pwd, 0)
		committee.Init()
		vite.miner.Init()
		vite.miner.Start()
		committee.Start()
	}
	vite.p2p.Start()

	fmt.Println("vite node start success you can find the runlog in", cfg.RunLogDir())
	return vite, nil
}

func (v *Vite) Ledger() handler_interface.Manager {
	return v.ledger
}

func (v *Vite) P2p() *p2p.Server {
	return v.p2p
}

func (v *Vite) Pm() protoInterface.ProtocolManager {
	return v.pm
}

func (v *Vite) WalletManager() *wallet.Manager {
	return v.walletManager
}

func (v *Vite) Signer() *signer.Master {
	return v.signer
}

//func (v *Vite) Config() *Config {
//	return v.config
//}

func (v *Vite) Miner() *miner.Miner {
	return v.miner
}
func (v *Vite) Verifier() consensus.Verifier {
	return v.verifier
}
