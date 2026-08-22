package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	router "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	xds "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	nodeID      = "grpc-client-node"
	clusterName = "user-service-cluster"
	targetName  = "user-service"
	backendIP   = "127.0.0.1"
	backendPort = 50055
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 1. Initialize Cache and Server xDS
	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	server := xds.NewServer(context.Background(), snapshotCache, nil)

	// 2. Define Backend (Endpoint)
	ep := &endpoint.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			Locality: &core.Locality{
				Region: "local-region",
				Zone:   "local-zone",
			},
			LoadBalancingWeight: wrapperspb.UInt32(1),
			LbEndpoints: []*endpoint.LbEndpoint{{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Address:       backendIP,
									PortSpecifier: &core.SocketAddress_PortValue{PortValue: backendPort},
								},
							},
						},
					},
				},
				HealthStatus: core.HealthStatus_HEALTHY,
			}},
		}},
	}

	// 3. Define Cluster
	cls := &cluster.Cluster{
		Name:                 clusterName,
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
		EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
			EdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{Ads: &core.AggregatedConfigSource{}},
			},
		},
	}

	// 4. Define Route and Listener for Client
	rt := &route.RouteConfiguration{
		Name: "local_route",
		VirtualHosts: []*route.VirtualHost{{
			Name:    "local_service",
			Domains: []string{targetName},
			Routes: []*route.Route{{
				Match: &route.RouteMatch{PathSpecifier: &route.RouteMatch_Prefix{Prefix: ""}},
				Action: &route.Route_Route{Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{Cluster: clusterName},
				}},
			}},
		}},
	}

	routerConfig, _ := anypb.New(&router.Router{})

	manager := &hcm.HttpConnectionManager{
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: rt,
		},
		HttpFilters: []*hcm.HttpFilter{
			{
				Name: "envoy.filters.http.router",
				ConfigType: &hcm.HttpFilter_TypedConfig{
					TypedConfig: routerConfig,
				},
			},
		},
	}
	managerAny, _ := anypb.New(manager)

	apiListener := &listener.ApiListener{
		ApiListener: managerAny,
	}

	ls := &listener.Listener{
		Name:        targetName,
		ApiListener: apiListener,
	}

	// 5. Push to Cache
	snap, _ := cache.NewSnapshot("1.0", map[resource.Type][]types.Resource{
		resource.EndpointType: {ep},
		resource.ClusterType:  {cls},
		resource.RouteType:    {rt},
		resource.ListenerType: {ls},
	})
	snapshotCache.SetSnapshot(context.Background(), nodeID, snap)

	// 6. Run xDS Server
	grpcServer := grpc.NewServer()
	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)

	lis, _ := net.Listen("tcp", ":15010")
	logger.Info("xDS Control Plane starting on :15010")
	grpcServer.Serve(lis)
}
