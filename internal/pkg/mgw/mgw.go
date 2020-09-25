/*
 *  Copyright (c) 2020, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *
 */
package mgw

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/fsnotify/fsnotify"
	"github.com/wso2/apictl/configs"
	mgwconfig "github.com/wso2/apictl/configs/confTypes"
	logger "github.com/wso2/apictl/loggers"
	apiserver "github.com/wso2/apictl/pkg/api"
	oasParser "github.com/wso2/apictl/pkg/oasparser"
)

var (
	debug       bool
	onlyLogging bool

	localhost = "0.0.0.0"

	port        uint
	gatewayPort uint
	alsPort     uint

	mode string

	version int32

	// cache cachev3.SnapshotCache
)

const (
	Rest = "rest"
)

func init() {
	flag.BoolVar(&debug, "debug", true, "Use debug logging")
	flag.BoolVar(&onlyLogging, "onlyLogging", false, "Only demo AccessLogging Service")
	// flag.UintVar(&port, "port", 18000, "Management server port")
	// flag.UintVar(&gatewayPort, "gateway", 18001, "Management server port for HTTP gateway")
	// flag.UintVar(&alsPort, "als", 18090, "Accesslog server port")
	// flag.StringVar(&mode, "ads", Ads, "Management server type (ads, xds, rest)")
}

// IDHash uses ID field as the node hash.
// type IDHash struct{}

// ID uses the node ID field
// func (IDHash) ID(node *corev3.Node) string {
// 	if node == nil {
// 		return "unknown"
// 	}
// 	return node.Id
// }

// var _ cachev3.NodeHash = IDHash{}

const grpcMaxConcurrentStreams = 1000000

// /**
//  * This starts an xDS server at the given port.
//  *
//  * @param ctx   Context
//  * @param server   Xds server instance
//  * @param port   Management server port
//  */
// func RunManagementServer(ctx context.Context, server xdsv3.Server, port uint) {
// 	var grpcOptions []grpc.ServerOption
// 	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams))
// 	grpcServer := grpc.NewServer()

// 	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
// 	if err != nil {
// 		logger.LoggerMgw.Fatal("failed to listen: ", err)
// 	}

// 	// register services
// 	discoveryv3.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
// 	endpointservicev3.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
// 	clusterservicev3.RegisterClusterDiscoveryServiceServer(grpcServer, server)
// 	routeservicev3.RegisterRouteDiscoveryServiceServer(grpcServer, server)
// 	listenerservicev3.RegisterListenerDiscoveryServiceServer(grpcServer, server)

// 	logger.LoggerMgw.Info("port: ", port, " management server listening")
// 	//log.Fatalf("", Serve(lis))
// 	//go func() {
// 	go func() {
// 		if err = grpcServer.Serve(lis); err != nil {
// 			logger.LoggerMgw.Error(err)
// 		}
// 	}()
// 	//<-ctx.Done()
// 	//grpcServer.GracefulStop()
// 	//}()

// }

/**
 * Recreate the envoy instances from swaggers.
 *
 * @param location   Swagger files location
 */
func updateEnvoy(location string) {
	// var nodeId string
	// if len(cache.GetStatusKeys()) > 0 {
	// 	nodeId = cache.GetStatusKeys()[0]
	// }

	envoystring := oasParser.GetProductionSources(location)

	// atomic.AddInt32(&version, 1)
	// logger.LoggerMgw.Infof(">>>>>>>>>>>>>>>>>>> creating snapshot Version " + fmt.Sprint(version))
	// snap := cachev3.NewSnapshot(fmt.Sprint(version), endpoints, clusters, routes, listeners, nil)
	// snap.Consistent()

	// err := cache.SetSnapshot(nodeId, snap)
	// if err != nil {
	// 	logger.LoggerMgw.Error(err)
	// }
	fmt.Print("\n\n" + envoystring + "\n\n")
}

/**
 * Run the management grpc server.
 *
 * @param conf  Swagger files location
 */
func Run(conf *mgwconfig.Config) {
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)
	watcher, _ := fsnotify.NewWatcher()
	err := watcher.Add(conf.Apis.Location)

	if err != nil {
		logger.LoggerMgw.Fatal("Error reading the api definitions.", err)
	}

	flag.Parse()

	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	//log config watcher
	watcherLogConf, _ := fsnotify.NewWatcher()
	errC := watcherLogConf.Add("resources/conf/log_config.toml")

	if errC != nil {
		logger.LoggerMgw.Fatal("Error reading the log configs. ", err)
	}

	// logger.LoggerMgw.Info("Starting control plane ....")

	// cache = cachev3.NewSnapshotCache(mode != Ads, IDHash{}, nil)

	// srv := xdsv3.NewServer(ctx, cache, nil)

	//als := &myals.AccessLogService{}
	//go RunAccessLogServer(ctx, als, alsPort)

	// start the xDS server
	// RunManagementServer(ctx, srv, port)
	go apiserver.Start(conf)

	updateEnvoy(conf.Apis.Location)
OUTER:
	for {
		select {
		case c := <-watcher.Events:
			switch c.Op.String() {
			case "WRITE":
				logger.LoggerMgw.Info("Loading updated swagger definition...")
				updateEnvoy(conf.Apis.Location)
			}
		case l := <-watcherLogConf.Events:
			switch l.Op.String() {
			case "WRITE":
				logger.LoggerMgw.Info("Loading updated log config file...")
				configs.ClearLogConfigInstance()
				logger.UpdateLoggers()
			}
		case s := <-sig:
			switch s {
			case os.Interrupt:
				logger.LoggerMgw.Info("Shutting down...")
				break OUTER
			}
		}
	}
	logger.LoggerMgw.Info("Bye!")
}
