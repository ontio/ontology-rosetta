/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/common/fdlimit"
	eventbus "github.com/ontio/ontology-eventbus/log"
	"github.com/ontio/ontology-rosetta/log"
	"github.com/ontio/ontology-rosetta/process"
	"github.com/ontio/ontology-rosetta/services"
	"github.com/ontio/ontology-rosetta/version"
	"github.com/ontio/ontology/cmd"
	"github.com/ontio/ontology/cmd/utils"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/config"
	nodelog "github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/genesis"
	"github.com/ontio/ontology/core/ledger"
	"github.com/ontio/ontology/events"
	"github.com/ontio/ontology/http/base/actor"
	"github.com/ontio/ontology/p2pserver"
	"github.com/ontio/ontology/p2pserver/actor/req"
	"github.com/ontio/ontology/txnpool"
	tc "github.com/ontio/ontology/txnpool/common"
	"github.com/ontio/ontology/txnpool/proc"
	"github.com/ontio/ontology/validator/stateful"
	"github.com/ontio/ontology/validator/stateless"
	"github.com/urfave/cli"
)

var disableLogFile bool

var (
	offlineFlag = cli.BoolFlag{
		Name:  "offline",
		Usage: "Run the Rosetta server in offline mode",
	}
	serverConfigFlag = cli.StringFlag{
		Name:  "server-config",
		Value: "./server-config.json",
		Usage: "Path to the config `<file>` for this Rosetta server",
	}
	validateStoreFlag = cli.BoolFlag{
		Name:  "validate-store",
		Usage: "Validate the indexed data in the Rosetta server's internal data store",
	}
)

type token struct {
	Contract string `json:"contract"`
	Decimals int32  `json:"decimals"`
	Symbol   string `json:"symbol"`
	Wasm     bool   `json:"wasm"`
}

type serverConfig struct {
	BlockWait  uint32   `json:"block_wait_seconds"`
	OEP4Tokens []*token `json:"oep4_tokens"`
	Port       uint32   `json:"port"`
	tokens     []*services.OEP4Token
	waitTime   time.Duration
}

func setupApp() *cli.App {
	app := cli.NewApp()
	app.Action = run
	app.Copyright = "Copyright (c) The Ontology Authors"
	app.Usage = "Ontology Rosetta Server"
	app.Version = fmt.Sprintf(
		"Ontology Rosetta Server: %s, Ontology Node: %s",
		version.Rosetta, version.Node,
	)
	app.Flags = []cli.Flag{
		// rosetta server settings
		serverConfigFlag,
		offlineFlag,
		validateStoreFlag,
		// base settings
		utils.ConfigFlag,
		utils.DataDirFlag,
		utils.DisableLogFileFlag,
		utils.LogDirFlag,
		utils.LogLevelFlag,
		utils.WasmVerifyMethodFlag,
		// txpool settings
		utils.DisableBroadcastNetTxFlag,
		utils.DisableSyncVerifyTxFlag,
		utils.GasLimitFlag,
		utils.GasPriceFlag,
		utils.TxpoolPreExecDisableFlag,
		// p2p settings
		utils.MaxConnInBoundFlag,
		utils.MaxConnInBoundForSingleIPFlag,
		utils.MaxConnOutBoundFlag,
		utils.NetworkIdFlag,
		utils.NodePortFlag,
		utils.ReservedPeersFileFlag,
		utils.ReservedPeersOnlyFlag,
	}
	app.Before = func(context *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return nil
	}
	// NOTE(tav): Unfortunately, cmd.flagGroup isn't exported, so we resort to
	// overriding the group for the import command as they are unused within
	// this application.
	idx := len(cmd.AppHelpFlagGroups) - 2
	group := cmd.AppHelpFlagGroups[idx]
	group.Name = "ROSETTA SERVER"
	group.Flags = []cli.Flag{
		serverConfigFlag,
		offlineFlag,
		validateStoreFlag,
	}
	cmd.AppHelpFlagGroups[idx] = group
	return app
}

func cliBool(ctx *cli.Context, flag cli.Flag) bool {
	return ctx.GlobalBool(utils.GetFlagName(flag))
}

func initLedger(ctx *cli.Context, cfg *config.OntologyConfig) *ledger.Ledger {
	events.Init()
	ldg, err := ledger.NewLedger(
		filepath.Join(cfg.Common.DataDir, cfg.P2PNode.NetworkName),
		config.GetStateHashCheckHeight(cfg.P2PNode.NetworkId),
	)
	if err != nil {
		log.Fatalf("Failed to open ledger: %s", err)
	}
	process.SetExitHandler(func() {
		log.Info("Closing ledger")
		if err := ldg.Close(); err != nil {
			log.Errorf("Failed to close ledger: %s", err)
		}
	})
	bk, err := cfg.GetBookkeepers()
	if err != nil {
		log.Fatalf("Failed to get the bookkeeper config: %s", err)
	}
	genesis, err := genesis.BuildGenesisBlock(bk, cfg.Genesis)
	if err != nil {
		log.Fatalf("Failed to build the genesis block: %s", err)
	}
	ledger.DefLedger = ldg
	if err := ldg.Init(bk, genesis); err != nil {
		log.Fatalf("Failed to initialize the ledger: %s", err)
	}
	log.Info("Ledger init success")
	return ldg
}

func initLog(ctx *cli.Context) {
	disableLogFile = cliBool(ctx, utils.DisableLogFileFlag)
	level := ctx.GlobalInt(utils.GetFlagName(utils.LogLevelFlag))
	log.InitLog(level, log.Stdout)
	if disableLogFile {
		nodelog.InitLog(level, nodelog.Stdout)
	} else {
		dir := ctx.GlobalString(utils.GetFlagName(utils.LogDirFlag))
		dir = filepath.Join(dir, "") + string(os.PathSeparator)
		eventbus.InitLog(dir)
		// NOTE(tav): We override the global PATH variable as it is used by
		// nodelog.CheckRotateLogFile when rotating log files.
		nodelog.PATH = dir
		nodelog.InitLog(level, dir, log.Stdout)
	}
}

func initNode(ctx *cli.Context, cfg *config.OntologyConfig, txpool *proc.TXPoolServer) *p2pserver.P2PServer {
	if cfg.Genesis.ConsensusType == config.CONSENSUS_TYPE_SOLO {
		return nil
	}
	node, err := p2pserver.NewServer(nil)
	if err != nil {
		log.Fatalf("Failed to create Node P2P server: %s", err)
	}
	err = node.Start()
	if err != nil {
		log.Fatalf("Failed to start the Node P2P service: %s", err)
	}
	txpool.Net = node.GetNetwork()
	req.SetTxnPoolPid(txpool.GetPID(tc.TxActor))
	actor.SetNetServer(node.GetNetwork())
	node.WaitForPeersStart()
	log.Info("Node init success")
	return node
}

func initNodeConfig(ctx *cli.Context) *config.OntologyConfig {
	log.Infof("Ontology Node version: %s", version.Node)
	log.Infof("Ontology Rosetta Server version: %s", version.Rosetta)
	cfg, err := cmd.SetOntologyConfig(ctx)
	if err != nil {
		log.Fatalf("Failed to process node config: %s", err)
	}
	return cfg
}

func initServer(ctx *cli.Context,
	cfg *config.OntologyConfig,
	scfg *serverConfig,
	node *p2pserver.P2PServer,
	offline bool,
) {
	store := initStore(cfg, scfg, offline)
	done := make(chan bool, 1)
	process.SetExitHandler(func() {
		if !offline {
			<-done
		}
		store.Close()
	})
	if !offline {
		ctx, cancel := context.WithCancel(context.Background())
		go store.IndexBlocks(ctx, services.IndexConfig{
			Done:     done,
			WaitTime: scfg.waitTime,
		})
		process.SetExitHandler(cancel)
	}
	router, err := services.Router(node, store, offline)
	if err != nil {
		log.Fatalf("Failed to load the Rosetta HTTP router: %s", err)
	}
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", scfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	log.Infof("Starting Rosetta Server on port %d", scfg.Port)
	go func() {
		process.SetExitHandler(func() {
			log.Info("Shutting down Rosetta HTTP Server gracefully")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := srv.Shutdown(ctx); err != nil {
				log.Errorf("Failed to shutdown Rosetta HTTP Server gracefully: %s", err)
			}
		})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Rosetta HTTP Server failed: %s", err)
		}
	}()
}

func initServerConfig(ctx *cli.Context) *serverConfig {
	path := ctx.GlobalString(utils.GetFlagName(serverConfigFlag))
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("Failed to open %q: %s", path, err)
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read %q: %s", path, err)
	}
	cfg := &serverConfig{}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		log.Fatalf("Failed to decode %q: %s", path, err)
	}
	if cfg.BlockWait == 0 {
		cfg.BlockWait = 1
	}
	cfg.waitTime = time.Duration(cfg.BlockWait) * time.Second
	for idx, token := range cfg.OEP4Tokens {
		if token.Contract == "" {
			log.Fatalf(
				`Missing "contract" field for OEP4 token at offset %d in %q`,
				idx, path,
			)
		}
		contract, err := common.AddressFromHexString(token.Contract)
		if err != nil {
			log.Fatalf(
				"Invalid OEP4 contract address %q found in %q: %s",
				token.Contract, path, err,
			)
		}
		if token.Decimals < 0 {
			log.Fatalf(
				`Invalid "decimals" value for OEP4 token at offset %d in %q: %d`,
				idx, path, token.Decimals,
			)
		}
		if token.Symbol == "" {
			log.Fatalf(
				`Missing "symbol" field for OEP4 token %q in %q`,
				token.Contract, path,
			)
		}
		cfg.tokens = append(cfg.tokens, &services.OEP4Token{
			Contract: contract,
			Decimals: token.Decimals,
			Symbol:   token.Symbol,
			Wasm:     token.Wasm,
		})
	}
	if cfg.Port > 65535 {
		log.Fatalf("Invalid port %d specified in %q", cfg.Port, path)
	}
	return cfg
}

func initStore(cfg *config.OntologyConfig, scfg *serverConfig, offline bool) *services.Store {
	store, err := services.NewStore(filepath.Join(
		cfg.Common.DataDir,
		cfg.P2PNode.NetworkName,
		"store",
	), scfg.tokens, offline)
	if err != nil {
		log.Fatalf("Unable to open the internal data store: %s", err)
	}
	return store
}

func initTxPool(ctx *cli.Context) *proc.TXPoolServer {
	actor.DisableSyncVerifyTx = cliBool(ctx, utils.DisableSyncVerifyTxFlag)
	txpool, err := txnpool.StartTxnPoolServer(
		cliBool(ctx, utils.TxpoolPreExecDisableFlag),
		cliBool(ctx, utils.DisableBroadcastNetTxFlag),
	)
	if err != nil {
		log.Fatalf("Failed to start the TxPool server: %s", err)
	}
	stlValidator, _ := stateless.NewValidator("stateless_validator")
	stlValidator.Register(txpool.GetPID(tc.VerifyRspActor))
	stlValidator2, _ := stateless.NewValidator("stateless_validator2")
	stlValidator2.Register(txpool.GetPID(tc.VerifyRspActor))
	stfValidator, _ := stateful.NewValidator("stateful_validator")
	stfValidator.Register(txpool.GetPID(tc.VerifyRspActor))
	actor.SetTxnPoolPid(txpool.GetPID(tc.TxPoolActor))
	actor.SetTxPid(txpool.GetPID(tc.TxActor))
	log.Info("TxPool init success")
	return txpool
}

func run(ctx *cli.Context) {
	initLog(ctx)
	setMaxOpenFiles()
	cfg := initNodeConfig(ctx)
	scfg := initServerConfig(ctx)
	if cliBool(ctx, validateStoreFlag) {
		runValidateStore(cfg, scfg)
	} else if cliBool(ctx, offlineFlag) {
		runOffline(ctx, cfg, scfg)
	} else {
		runOnline(ctx, cfg, scfg)
	}
}

func runOffline(ctx *cli.Context, cfg *config.OntologyConfig, scfg *serverConfig) {
	initServer(ctx, cfg, scfg, &p2pserver.P2PServer{}, true)
	select {}
}

func runOnline(ctx *cli.Context, cfg *config.OntologyConfig, scfg *serverConfig) {
	ldg := initLedger(ctx, cfg)
	txpool := initTxPool(ctx)
	node := initNode(ctx, cfg, txpool)
	initServer(ctx, cfg, scfg, node, false)
	ticker := time.NewTicker(config.DEFAULT_GEN_BLOCK_TIME * time.Second)
	for range ticker.C {
		log.Infof("CurrentBlockHeight = %d", ldg.GetCurrentBlockHeight())
		if !disableLogFile {
			nodelog.CheckRotateLogFile()
		}
	}
}

func runValidateStore(cfg *config.OntologyConfig, scfg *serverConfig) {
	ctx := context.Background()
	store := initStore(cfg, scfg, false)
	log.Info("Started indexing any missing blocks")
	store.IndexBlocks(ctx, services.IndexConfig{
		ExitEarly: true,
		WaitTime:  scfg.waitTime,
	})
	log.Info("Finished indexing blocks")
}

func setMaxOpenFiles() {
	max, err := fdlimit.Maximum()
	if err != nil {
		log.Errorf("Failed to get fdlimit: %s", err)
		return
	}
	_, err = fdlimit.Raise(uint64(max))
	if err != nil {
		log.Errorf("Failed to raise fdlimit: %s", err)
		return
	}
}

func main() {
	if err := setupApp().Run(os.Args); err != nil {
		cmd.PrintErrorMsg(err.Error())
		os.Exit(1)
	}
}
