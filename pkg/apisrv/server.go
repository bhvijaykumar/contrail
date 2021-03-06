package apisrv

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	protocodec "github.com/gogo/protobuf/codec"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/Juniper/contrail/pkg/apisrv/client"
	apicommon "github.com/Juniper/contrail/pkg/apisrv/common"
	"github.com/Juniper/contrail/pkg/apisrv/discovery"
	"github.com/Juniper/contrail/pkg/apisrv/keystone"
	"github.com/Juniper/contrail/pkg/apisrv/replication"
	"github.com/Juniper/contrail/pkg/collector"
	"github.com/Juniper/contrail/pkg/collector/analytics"
	"github.com/Juniper/contrail/pkg/constants"
	"github.com/Juniper/contrail/pkg/db"
	"github.com/Juniper/contrail/pkg/db/cache"
	etcdclient "github.com/Juniper/contrail/pkg/db/etcd"
	"github.com/Juniper/contrail/pkg/fileutil"
	"github.com/Juniper/contrail/pkg/logutil"
	"github.com/Juniper/contrail/pkg/models"
	"github.com/Juniper/contrail/pkg/neutron"
	"github.com/Juniper/contrail/pkg/services"
	"github.com/Juniper/contrail/pkg/types"
)

// Server HTTP paths.
const (
	FQNameToIDPath  = "fqname-to-id"
	IDToFQNamePath  = "id-to-fqname"
	UserAgentKVPath = "useragent-kv"
	WatchPath       = "watch"
)

//TODO(nati) use parameter
var extensions = []func(server *Server) error{}

//RegisterExtension registers extension callback.
func RegisterExtension(f func(server *Server) error) {
	extensions = append(extensions, f)
}

//Server represents Intent API Server.
type Server struct {
	Echo                       *echo.Echo
	GRPCServer                 *grpc.Server
	Keystone                   *keystone.Keystone
	DBService                  *db.Service
	Proxy                      *proxyService
	Service                    services.Service
	IPAMServer                 services.IPAMServer
	ChownServer                services.ChownServer
	SetTagServer               services.SetTagServer
	RefRelaxServer             services.RefRelaxServer
	UserAgentKVServer          services.UserAgentKVServer
	FQNameToIDServer           services.FQNameToIDServer
	IDToFQNameServer           services.IDToFQNameServer
	PropCollectionUpdateServer services.PropCollectionUpdateServer
	Cache                      *cache.DB
	Collector                  collector.Collector
	VNCReplicator              *replication.Replicator
}

// NewServer makes a server
func NewServer() (*Server, error) {
	server := &Server{
		Echo: echo.New(),
	}
	return server, nil
}

//Application with custom logics can embed server struct, and overwrite
//this method.
func (s *Server) setupService() (*services.ContrailService, error) {
	var serviceChain []services.Service

	cs, err := s.contrailService()
	if err != nil {
		return nil, err
	}
	serviceChain = append(serviceChain, cs)

	serviceChain = append(serviceChain, &services.RefUpdateToUpdateService{
		ReadService:       s.DBService,
		InTransactionDoer: s.DBService,
	})

	serviceChain = append(serviceChain, &services.SanitizerService{
		MetadataGetter: s.DBService,
	})

	serviceChain = append(serviceChain, &services.RBACService{
		ReadService: s.DBService,
		AAAMode:     viper.GetString("aaa_mode")})

	if viper.GetBool("server.enable_vnc_neutron") {
		serviceChain = append(serviceChain, &neutron.Service{
			Keystone: &client.Keystone{
				URL: viper.GetString("keystone.authurl"),
				HTTPClient: &http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: viper.GetBool("keystone.insecure")},
					},
				},
			},
			ReadService:    s.DBService,
			MetadataGetter: s.DBService,
			WriteService: &services.InternalContextWriteServiceWrapper{
				WriteService: serviceChain[0],
			},
			InTransactionDoer: s.DBService,
		})
	}

	serviceChain = append(serviceChain, &types.ContrailTypeLogicService{
		ReadService:       s.DBService,
		InTransactionDoer: s.DBService,
		AddressManager:    s.DBService,
		IntPoolAllocator:  s.DBService,
		MetadataGetter:    s.DBService,
		WriteService: &services.InternalContextWriteServiceWrapper{
			WriteService: serviceChain[0],
		},
	})

	serviceChain = append(serviceChain, services.NewQuotaCheckerService(s.DBService))

	if viper.GetBool("server.notify_etcd") {
		en := etcdNotifier()
		if en != nil {
			serviceChain = append(serviceChain, en)
		}
	}

	serviceChain = append(serviceChain, s.DBService)

	services.Chain(serviceChain...)
	return cs, nil
}

func (s *Server) contrailService() (*services.ContrailService, error) {
	tv, err := models.NewTypeValidatorWithFormat()
	if err != nil {
		return nil, err
	}

	cs := &services.ContrailService{
		BaseService:        services.BaseService{},
		DBService:          s.DBService,
		TypeValidator:      tv,
		MetadataGetter:     s.DBService,
		InTransactionDoer:  s.DBService,
		IntPoolAllocator:   s.DBService,
		RefRelaxer:         s.DBService,
		UserAgentKVService: s.DBService,
		Collector:          s.Collector,
	}

	cs.RegisterRESTAPI(s.Echo)
	return cs, nil
}

func etcdNotifier() services.Service {
	// TODO(Michał): Make the codec configurable
	en, err := etcdclient.NewNotifierService(viper.GetString(constants.ETCDPathVK), models.JSONCodec)
	if err != nil {
		logrus.WithError(err).Error("Failed to add etcd Notifier Service - ignoring")
		return nil
	}
	return en
}

func (s *Server) serveDynamicProxy(endpointStore *apicommon.EndpointStore) {
	s.Proxy = newProxyService(s.Echo, endpointStore, s.DBService)
	s.Proxy.serve()
}

func (s *Server) startVNCReplicator(endpointStore *apicommon.EndpointStore) (err error) {
	s.VNCReplicator, err = replication.New(endpointStore)
	if err != nil {
		return err
	}
	err = s.VNCReplicator.Start()
	if err != nil {
		return err
	}
	return nil
}

//Init setup the server.
// nolint: gocyclo
func (s *Server) Init() (err error) {
	// TODO (Kamil): should we refactor server to use a local logger?
	if err = logutil.Configure(viper.GetString("log_level")); err != nil {
		return err
	}

	// TODO: integrate Echo's logger with logrus
	e := s.Echo
	if viper.GetBool("server.log_api") {
		e.Use(middleware.Logger())
	} else {
		e.Logger.SetOutput(ioutil.Discard) // Disables Echo's built-in logging.
	}

	if viper.GetBool("server.log_body") {
		e.Use(middleware.BodyDump(func(c echo.Context, requestBody, responseBody []byte) {
			logrus.WithFields(logrus.Fields{
				"request-body":  string(requestBody),
				"response-body": string(responseBody),
			}).Debug("Request handled")
		}))
	}

	if viper.GetBool("server.enable_gzip") {
		e.Use(middleware.Gzip())
	}

	e.Use(middleware.Recover())
	e.Binder = &customBinder{}

	s.DBService, err = db.NewServiceFromConfig()
	if err != nil {
		return err
	}

	if err = s.setupCollector(); err != nil {
		return err
	}

	cs, err := s.setupService()
	if err != nil {
		return err
	}

	s.Service = cs
	s.IPAMServer = cs
	s.ChownServer = cs
	s.SetTagServer = cs
	s.RefRelaxServer = cs
	s.UserAgentKVServer = cs
	s.FQNameToIDServer = cs
	s.IDToFQNameServer = cs
	s.PropCollectionUpdateServer = cs

	if viper.GetBool("server.enable_vnc_neutron") {
		s.setupNeutronService(cs)
	}

	readTimeout := viper.GetInt("server.read_timeout")
	writeTimeout := viper.GetInt("server.write_timeout")
	e.Server.ReadTimeout = time.Duration(readTimeout) * time.Second
	e.Server.WriteTimeout = time.Duration(writeTimeout) * time.Second

	cors := viper.GetString("server.cors")

	if cors != "" {
		logrus.Printf("Enabling CORS for %s", cors)
		if cors == "*" {
			logrus.Printf("cors for * have security issue. DO NOT USE THIS IN PRODUCTION")
		}
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:  []string{cors},
			AllowMethods:  []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
			AllowHeaders:  []string{"X-Auth-Token", "Content-Type"},
			ExposeHeaders: []string{"X-Total-Count"},
		}))
	}

	staticPath := viper.GetStringMapString("server.static_files")
	for prefix, root := range staticPath {
		e.Static(prefix, root)
	}

	proxy := viper.GetStringMapStringSlice("server.proxy")
	for prefix, target := range proxy {
		g := e.Group(prefix)
		g.Use(removePathPrefixMiddleware(prefix))

		t, err := url.Parse(target[0])
		if err != nil {
			return errors.Wrapf(err, "bad proxy target URL: %s", target[0])
		}
		g.Use(proxyMiddleware(t, viper.GetBool("server.proxy.insecure")))
	}
	// serve dynamic proxy based on configured endpoints
	endpointStore := apicommon.MakeEndpointStore() // sync map to store proxy endpoints
	s.serveDynamicProxy(endpointStore)

	if viper.GetBool("server.enable_vnc_replication") {
		if err := s.startVNCReplicator(endpointStore); err != nil {
			return err
		}
	}

	keystoneAuthURL := viper.GetString("keystone.authurl")
	var keystoneClient *keystone.Client
	if keystoneAuthURL != "" {
		keystoneClient = keystone.NewKeystoneClient(keystoneAuthURL, viper.GetBool("keystone.insecure"))
		skipPaths, err := keystone.GetAuthSkipPaths()
		if err != nil {
			return errors.Wrap(err, "failed to setup paths skipped from authentication")
		}
		e.Use(keystone.AuthMiddleware(keystoneClient, skipPaths, endpointStore))
	} else if viper.GetString("auth_type") == "no-auth" {
		e.Use(noAuthMiddleware())
	}
	localKeystone := viper.GetBool("keystone.local")
	if localKeystone {
		k, err := keystone.Init(e, endpointStore, keystoneClient)
		if err != nil {
			return errors.Wrap(err, "Failed to init local keystone server")
		}
		s.Keystone = k
	}

	if viper.GetBool("server.enable_grpc") {
		if !viper.GetBool("server.tls.enabled") {
			return errors.New("GRPC support requires TLS configuration")
		}
		logrus.Debug("Enabling gRPC server")
		opts := []grpc.ServerOption{
			// TODO(Michal): below option potentially breaks compatibility for non golang grpc clients.
			// Ensure it doesn't or find a better solution for un/marshaling `oneof` fields properly.
			grpc.CustomCodec(protocodec.New(0)),
		}
		if keystoneAuthURL != "" {
			opts = append(opts, grpc.UnaryInterceptor(keystone.AuthInterceptor(keystoneClient, endpointStore)))
		} else if viper.GetBool("no_auth") {
			opts = append(opts, grpc.UnaryInterceptor(noAuthInterceptor()))
		}
		s.GRPCServer = grpc.NewServer(opts...)
		services.RegisterContrailServiceServer(s.GRPCServer, s.Service)
		services.RegisterIPAMServer(s.GRPCServer, s.IPAMServer)
		services.RegisterChownServer(s.GRPCServer, s.ChownServer)
		services.RegisterSetTagServer(s.GRPCServer, s.SetTagServer)
		services.RegisterRefRelaxServer(s.GRPCServer, s.RefRelaxServer)
		services.RegisterFQNameToIDServer(s.GRPCServer, s.FQNameToIDServer)
		services.RegisterIDToFQNameServer(s.GRPCServer, s.IDToFQNameServer)
		services.RegisterUserAgentKVServer(s.GRPCServer, s.UserAgentKVServer)
		services.RegisterPropCollectionUpdateServer(s.GRPCServer, s.PropCollectionUpdateServer)
		e.Use(gRPCMiddleware(s.GRPCServer))
	}

	if viper.GetBool("homepage.enabled") {
		s.setupHomepage()
	}

	s.setupWatchAPI()
	s.setupActionResources(cs)

	if viper.GetBool("recorder.enabled") {
		file := viper.GetString("recorder.file")
		scenario := &struct {
			Workflow []*recorderTask `yaml:"workflow,omitempty"`
		}{}
		var mutex sync.Mutex
		e.Use(middleware.BodyDump(func(c echo.Context, requestBody, responseBody []byte) {
			var data interface{}
			err := json.Unmarshal(requestBody, &data)
			if err != nil {
				logrus.Debug("malformed json input")
			}
			var expected interface{}
			err = json.Unmarshal(responseBody, &expected)
			if err != nil {
				logrus.Debug("malformed json response")
			}
			task := &recorderTask{
				Request: &client.Request{
					Method:   c.Request().Method,
					Path:     c.Request().URL.Path,
					Expected: []int{c.Response().Status},
					Data:     data,
				},
				Expect: expected,
			}
			mutex.Lock()
			defer mutex.Unlock()
			scenario.Workflow = append(scenario.Workflow, task)
			err = fileutil.SaveFile(file, scenario)
			if err != nil {
				logrus.Warn(err)
			}
		}))
	}

	return s.applyExtensions()
}

func (s *Server) setupNeutronService(cs services.Service) *neutron.Server {
	n := &neutron.Server{
		ReadService:       cs,
		WriteService:      cs,
		UserAgentKV:       s.UserAgentKVServer,
		IDToFQNameService: s.IDToFQNameServer,
		FQNameToIDService: s.FQNameToIDServer,
		InTransactionDoer: s.DBService,
		Log:               logutil.NewLogger("neutron-server"),
	}
	n.RegisterNeutronAPI(s.Echo)
	return n
}

type recorderTask struct {
	Request *client.Request `yaml:"request,omitempty"`
	Expect  interface{}     `yaml:"expect,omitempty"`
}

func (s *Server) applyExtensions() error {
	for _, extension := range extensions {
		err := extension(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) setupHomepage() {
	dh := discovery.NewHandler()

	services.RegisterSingularPaths(func(path string, name string) {
		dh.Register(path, "", name, "resource-base")
	})
	services.RegisterPluralPaths(func(path string, name string) {
		dh.Register(path, "", name, "collection")
	})

	dh.Register(FQNameToIDPath, "POST", "name-to-id", "action")
	dh.Register(IDToFQNamePath, "POST", "id-to-name", "action")
	dh.Register(UserAgentKVPath, "POST", UserAgentKVPath, "action")
	dh.Register(services.RefUpdatePath, "POST", services.RefUpdatePath, "action")
	dh.Register(services.RefRelaxForDeletePath, "POST", services.RefRelaxForDeletePath, "action")
	dh.Register(services.PropCollectionUpdatePath, "POST", services.PropCollectionUpdatePath, "action")
	dh.Register(services.SetTagPath, "POST", services.SetTagPath, "action")
	dh.Register(services.ChownPath, "POST", services.ChownPath, "action")
	dh.Register(services.IntPoolPath, "GET", services.IntPoolPath, "action")
	dh.Register(services.IntPoolPath, "POST", services.IntPoolPath, "action")
	dh.Register(services.IntPoolPath, "DELETE", services.IntPoolPath, "action")
	dh.Register(services.IntPoolsPath, "POST", services.IntPoolsPath, "action")
	dh.Register(services.IntPoolsPath, "DELETE", services.IntPoolsPath, "action")
	dh.Register(services.ObjPerms, "GET", services.ObjPerms, "action")

	// TODO: register sync?

	// TODO action resources
	// TODO documentation
	// TODO VN IP alloc
	// TODO VN IP free
	// TODO subnet IP count
	// TODO security policy draft

	s.Echo.GET("/", dh.Handle)
}

func (s *Server) setupWatchAPI() {
	if !viper.GetBool("cache.enabled") {
		return
	}
	s.Echo.GET(WatchPath, s.watchHandler)
}

func (s *Server) setupActionResources(cs *services.ContrailService) {
	s.Echo.POST(FQNameToIDPath, cs.RESTFQNameToUUID)
	s.Echo.POST(IDToFQNamePath, cs.RESTIDToFQName)
	s.Echo.POST(UserAgentKVPath, cs.RESTUserAgentKV)
}

func (s *Server) setupCollector() error {
	cfg := &analytics.Config{}
	var err error
	if err = viper.UnmarshalKey("collector", cfg); err != nil {
		return errors.Wrap(err, "failed to unmarshal collector config")
	}
	if s.Collector, err = analytics.NewCollector(cfg); err != nil {
		return errors.Wrap(err, "failed to create collector")
	}
	analytics.AddLoggerHook(s.Collector)
	s.Echo.Use(middleware.BodyDump(func(
		ctx echo.Context, reqBody, resBody []byte,
	) {
		s.Collector.Send(analytics.RESTAPITrace(ctx, reqBody, resBody))
	}))
	return nil
}

// Run runs Server.
func (s *Server) Run() error {
	defer func() {
		if err := s.Close(); err != nil {
			logrus.WithError(err).Info("Closing DBService failed")
		}
	}()

	if viper.GetBool("server.tls.enabled") {
		return s.Echo.StartTLS(
			viper.GetString("server.address"),
			viper.GetString("server.tls.cert_file"),
			viper.GetString("server.tls.key_file"),
		)
	}

	return s.Echo.Start(viper.GetString("server.address"))
}

// Close closes Server resources.
func (s *Server) Close() error {
	if s.VNCReplicator != nil {
		s.VNCReplicator.Stop()
	}
	s.Proxy.stop()
	return s.DBService.Close()
}

// DB returns sql.DB handle.
func (s *Server) DB() *sql.DB {
	return s.DBService.DB()
}
