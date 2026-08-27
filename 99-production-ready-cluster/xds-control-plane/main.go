package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

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
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	k8scache "k8s.io/client-go/tools/cache"
)

const (
	grpcPort                 = ":15010"
	grpcMaxConcurrentStreams = 1000000
	nodeID                   = "grpc-client-node"
)

var (
	version      int64
	stateMutex   sync.Mutex
	endpointsMap = make(map[string][]string) // Store IPs for all services
)

func main() {
	ctx := context.Background()
	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	xdsServer := xds.NewServer(ctx, snapshotCache, nil)

	go startK8sWatcher(snapshotCache)

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams))
	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, xdsServer)

	log.Printf("xDS Control Plane (Full Mesh) listening on %s", grpcPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}
}

func startK8sWatcher(xdsCache cache.SnapshotCache) {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to get k8s config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create K8s client: %v", err)
	}

	factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)
	endpointsInformer := factory.Core().V1().Endpoints().Informer()

	endpointsInformer.AddEventHandler(k8scache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { handleEndpointUpdate(obj.(*corev1.Endpoints), xdsCache) },
		UpdateFunc: func(old, new interface{}) { handleEndpointUpdate(new.(*corev1.Endpoints), xdsCache) },
		DeleteFunc: func(obj interface{}) { handleEndpointUpdate(obj.(*corev1.Endpoints), xdsCache) },
	})

	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
}

func handleEndpointUpdate(ep *corev1.Endpoints, xdsCache cache.SnapshotCache) {
	serviceName := ep.Name
	if serviceName != "order-service" && serviceName != "user-service" {
		return
	}

	var activeIPs []string
	for _, subset := range ep.Subsets {
		for _, addr := range subset.Addresses {
			activeIPs = append(activeIPs, addr.IP)
		}
	}

	// Lock state while rebuilding the global snapshot
	stateMutex.Lock()
	defer stateMutex.Unlock()

	endpointsMap[serviceName] = activeIPs
	log.Printf("Detected %d pods for %s. Rebuilding xDS global snapshot...", len(activeIPs), serviceName)

	var listeners []types.Resource
	var routes []types.Resource
	var clusters []types.Resource
	var endpoints []types.Resource

	// Build resources for ALL tracked services
	for srvName, ips := range endpointsMap {
		if len(ips) == 0 {
			continue // Skip if no healthy pods
		}
		listeners = append(listeners, buildListener(srvName))
		routes = append(routes, buildRoute(srvName))
		clusters = append(clusters, buildCluster(srvName))
		endpoints = append(endpoints, buildEndpoints(srvName, ips))
	}

	v := atomic.AddInt64(&version, 1)
	versionStr := fmt.Sprintf("v%d", v)

	snap, _ := cache.NewSnapshot(versionStr, map[resource.Type][]types.Resource{
		resource.ListenerType: listeners,
		resource.RouteType:    routes,
		resource.ClusterType:  clusters,
		resource.EndpointType: endpoints,
	})

	if err := xdsCache.SetSnapshot(context.Background(), nodeID, snap); err != nil {
		log.Printf("Failed to set snapshot: %v", err)
	}
}

// 1. Build Listener (LDS) - Uses ApiListener for gRPC
func buildListener(serviceName string) *listener.Listener {
	routeName := serviceName + "-route"
	routerConfig, _ := anypb.New(&router.Router{})

	manager := &hcm.HttpConnectionManager{
		CodecType: hcm.HttpConnectionManager_AUTO,
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource: &core.ConfigSource{
					ConfigSourceSpecifier: &core.ConfigSource_Ads{
						Ads: &core.AggregatedConfigSource{},
					},
				},
				RouteConfigName: routeName,
			},
		},
		HttpFilters: []*hcm.HttpFilter{
			{
				Name: wellknown.Router,
				ConfigType: &hcm.HttpFilter_TypedConfig{
					TypedConfig: routerConfig,
				},
			},
		},
	}
	pbst, _ := anypb.New(manager)

	return &listener.Listener{
		Name: serviceName, // Must match the dial target: xds:///order-service
		ApiListener: &listener.ApiListener{
			ApiListener: pbst,
		},
	}
}

// 2. Build Route (RDS)
func buildRoute(serviceName string) *route.RouteConfiguration {
	return &route.RouteConfiguration{
		Name: serviceName + "-route",
		VirtualHosts: []*route.VirtualHost{{
			Name:    serviceName,
			Domains: []string{"*"}, // Accept any domain
			Routes: []*route.Route{{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{Prefix: "/"},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{Cluster: serviceName},
					},
				},
			}},
		}},
	}
}

// 3. Build Cluster (CDS)
func buildCluster(serviceName string) *cluster.Cluster {
	return &cluster.Cluster{
		Name:                 serviceName,
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
		EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
			EdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
			},
		},
	}
}

// 4. Build Endpoints (EDS)
func buildEndpoints(serviceName string, ips []string) *endpoint.ClusterLoadAssignment {
	var lbEndpoints []*endpoint.LbEndpoint
	port := uint32(50068)
	if serviceName == "user-service" {
		port = 50069
	}

	for _, ip := range ips {
		lbEndpoints = append(lbEndpoints, &endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol:      core.SocketAddress_TCP,
								Address:       ip,
								PortSpecifier: &core.SocketAddress_PortValue{PortValue: port},
							},
						},
					},
				},
			},
			HealthStatus: core.HealthStatus_HEALTHY,
		})
	}

	return &endpoint.ClusterLoadAssignment{
		ClusterName: serviceName,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			Locality:            &core.Locality{Region: "minikube"},
			LoadBalancingWeight: wrapperspb.UInt32(1),
			LbEndpoints:         lbEndpoints,
		}},
	}
}
