package main

import (
	"context"
	rpctypes "github.com/filecoin-project/lotus/extern/miningstate/types"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	mux "github.com/gorilla/mux"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-jsonrpc/auth"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/apistruct"
	"github.com/filecoin-project/lotus/build"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/lib/ulimit"
	"github.com/filecoin-project/lotus/node"
	"github.com/filecoin-project/lotus/node/impl"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"
)

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start a lotus miner process",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "api",
			Usage: "2345",
		},
		&cli.BoolFlag{
			Name:  "enable-gpu-proving",
			Usage: "enable use of GPU for mining operations",
			Value: true,
		},
		&cli.BoolFlag{
			Name:  "nosync",
			Usage: "don't check full-node sync status",
		},
		&cli.BoolFlag{
			Name:  "manage-fdlimit",
			Usage: "manage open file limit",
			Value: true,
		},
		&cli.StringFlag{
			Name:  "rpc-address-ma",
			Usage: "urrpc server address eg 0.0.0.0:5977 ",
		},
		&cli.StringFlag{
			Name:  "rpc-address-local",
			Usage: "urrpc server address eg 0.0.0.0:5977 ",
		},
		&cli.StringFlag{
			Name:  "rpc-address-role",
			Usage: "Master/SLAVE",
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Bool("enable-gpu-proving") {
			err := os.Setenv("BELLMAN_NO_GPU", "true")
			if err != nil {
				return err
			}
		}

		nodeApi, ncloser, err := lcli.GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer ncloser()
		ctx := lcli.DaemonContext(cctx)

		v, err := nodeApi.Version(ctx)
		if err != nil {
			return err
		}

		if cctx.Bool("manage-fdlimit") {
			if _, _, err := ulimit.ManageFdLimit(); err != nil {
				log.Errorf("setting file descriptor limit: %s", err)
			}
		}


		if v.APIVersion != build.FullAPIVersion {
			return xerrors.Errorf("lotus-daemon API version doesn't match: expected: %s", api.Version{APIVersion: build.FullAPIVersion})
        }
		
		var RpcAddressForLocal rpctypes.LocalServerAddr
		var RpcAddressForUM rpctypes.RemoteServerAddr
		var Role rpctypes.Role
		RoleRpc := cctx.String("rpc-address-role")
		if RoleRpc != ""{
			if RoleRpc == "Master"{
				Role = rpctypes.Role_Master
				RpcAddressForLocal = rpctypes.LocalServerAddr(cctx.String("rpc-address-local"))
				RpcAddressForUM = rpctypes.RemoteServerAddr(cctx.String("rpc-address-ma"))
				log.Info("###########[Master]############Local=",RpcAddressForLocal)
			}else if RoleRpc == "Slave"{
				Role = rpctypes.Role_Slave
				RpcAddressForLocal = rpctypes.LocalServerAddr(cctx.String("rpc-address-local"))
				RpcAddressForUM = rpctypes.RemoteServerAddr(cctx.String("rpc-address-ma"))
				log.Info("############[Slave]###########Local=",RpcAddressForLocal,"Connect->",RpcAddressForUM)
			}
		}

		log.Info("Checking full node sync status")

		if !cctx.Bool("nosync") {
			if err := lcli.SyncWait(ctx, nodeApi); err != nil {
				return xerrors.Errorf("sync wait: %w", err)
			}
		}

		minerRepoPath := cctx.String(FlagMinerRepo)
		r, err := repo.NewFS(minerRepoPath)
		if err != nil {
			return err
		}

		ok, err := r.Exists()
		if err != nil {
			return err
		}
		if !ok {
			return xerrors.Errorf("repo at '%s' is not initialized, run 'lotus-miner init' to set it up", minerRepoPath)
		}

		shutdownChan := make(chan struct{})

		var minerapi api.StorageMiner
		stop, err := node.New(ctx,
			node.StorageMiner(&minerapi),
			// UM parms
			node.Override(new(rpctypes.Role),Role),
			node.Override(new(rpctypes.LocalServerAddr),RpcAddressForLocal),
			node.Override(new(rpctypes.RemoteServerAddr),RpcAddressForUM),

			node.Override(new(dtypes.ShutdownChan), shutdownChan),
			node.Online(),
			node.Repo(r),

			node.ApplyIf(func(s *node.Settings) bool { return cctx.IsSet("api") },
				node.Override(new(dtypes.APIEndpoint), func() (dtypes.APIEndpoint, error) {
					return multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/" + cctx.String("api"))
				})),
			node.Override(new(api.FullNode), nodeApi),
		)
		if err != nil {
			return err
		}

		endpoint, err := r.APIEndpoint()
		if err != nil {
			return err
		}

		// Bootstrap with full node
		remoteAddrs, err := nodeApi.NetAddrsListen(ctx)
		if err != nil {
			return err
		}

		if err := minerapi.NetConnect(ctx, remoteAddrs); err != nil {
			return err
		}

		log.Infof("Remote version %s", v)

		lst, err := manet.Listen(endpoint)
		if err != nil {
			return xerrors.Errorf("could not listen: %w", err)
		}

		mux := mux.NewRouter()

		rpcServer := jsonrpc.NewServer()
		rpcServer.Register("Filecoin", apistruct.PermissionedStorMinerAPI(minerapi))

		mux.Handle("/rpc/v0", rpcServer)
		mux.PathPrefix("/remote").HandlerFunc(minerapi.(*impl.StorageMinerAPI).ServeRemote)
		mux.PathPrefix("/").Handler(http.DefaultServeMux) // pprof

		ah := &auth.Handler{
			Verify: minerapi.AuthVerify,
			Next:   mux.ServeHTTP,
		}

		srv := &http.Server{Handler: ah}

		sigChan := make(chan os.Signal, 2)
		go func() {
			select {
			case sig := <-sigChan:
				log.Warnw("received shutdown", "signal", sig)
			case <-shutdownChan:
				log.Warn("received shutdown")
			}

			log.Warn("Shutting down...")
			if err := stop(context.TODO()); err != nil {
				log.Errorf("graceful shutting down failed: %s", err)
			}
			if err := srv.Shutdown(context.TODO()); err != nil {
				log.Errorf("shutting down RPC server failed: %s", err)
			}
			log.Warn("Graceful shutdown successful")
		}()
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

		return srv.Serve(manet.NetListener(lst))
	},
}
