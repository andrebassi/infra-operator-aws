package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/controllers"
	"infra-operator/pkg/api"
	"infra-operator/pkg/cli"
	"infra-operator/pkg/clients"
	"infra-operator/pkg/core"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(infrav1alpha1.AddToScheme(scheme))
}

func main() {
	// Check if running in CLI mode
	if cli.IsCliCommand(os.Args) {
		cli.Main()
		return
	}

	// Check if running in API server mode
	if isServeCommand(os.Args) {
		runAPIServer()
		return
	}

	// Operator mode
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var useCleanArchitecture bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&useCleanArchitecture, "clean-arch", true,
		"Use Clean Architecture implementation (default: true)")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "infra-operator.aws-infra-operator.runner.codes",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// ========================================
	// Dependency Injection (Clean Architecture)
	// ========================================

	// Create AWS client factory (shared across all controllers)
	awsClientFactory := clients.NewAWSClientFactory(mgr.GetClient())

	// Setup AWSProvider Controller (no changes needed - already clean)
	if err = (&controllers.AWSProviderReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AWSProvider")
		os.Exit(1)
	}

	// Setup S3Bucket Controller
	if useCleanArchitecture {
		// Use Clean Architecture version
		setupLog.Info("Using Clean Architecture implementation")

		if err = (&controllers.S3BucketReconcilerClean{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "S3BucketClean")
			os.Exit(1)
		}

		// Setup DynamoDBTable Controller
		if err = (&controllers.DynamoDBTableReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "DynamoDBTable")
			os.Exit(1)
		}

		// Setup SQSQueue Controller
		if err = (&controllers.SQSQueueReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "SQSQueue")
			os.Exit(1)
		}

		// Setup SNSTopic Controller
		if err = (&controllers.SNSTopicReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "SNSTopic")
			os.Exit(1)
		}

		// Setup LambdaFunction Controller
		if err = (&controllers.LambdaFunctionReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "LambdaFunction")
			os.Exit(1)
		}

		// Setup RDSInstance Controller
		if err = (&controllers.RDSInstanceReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "RDSInstance")
			os.Exit(1)
		}

		// Setup ECRRepository Controller
		if err = (&controllers.ECRRepositoryReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ECRRepository")
			os.Exit(1)
		}

		// Setup IAMRole Controller
		if err = (&controllers.IAMRoleReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "IAMRole")
			os.Exit(1)
		}

		// Setup SecretsManagerSecret Controller
		if err = (&controllers.SecretsManagerSecretReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "SecretsManagerSecret")
			os.Exit(1)
		}

		// Setup KMSKey Controller
		if err = (&controllers.KMSKeyReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "KMSKey")
			os.Exit(1)
		}

		// Setup EC2Instance Controller
		if err = (&controllers.EC2InstanceReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "EC2Instance")
			os.Exit(1)
		}

		// Setup ElastiCacheCluster Controller
		if err = (&controllers.ElastiCacheClusterReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ElastiCacheCluster")
			os.Exit(1)
		}

		// Setup VPC Controller
		if err = (&controllers.VPCReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "VPC")
			os.Exit(1)
		}

		// Setup Subnet Controller
		if err = (&controllers.SubnetReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Subnet")
			os.Exit(1)
		}

		// Setup InternetGateway Controller
		if err = (&controllers.InternetGatewayReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "InternetGateway")
			os.Exit(1)
		}

		// Setup NATGateway Controller
		if err = (&controllers.NATGatewayReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "NATGateway")
			os.Exit(1)
		}

		// Setup SecurityGroup Controller
		if err = (&controllers.SecurityGroupReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			AWSClientFactory: awsClientFactory,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "SecurityGroup")
			os.Exit(1)
		}
	} else {
		// Use original implementation (legacy)
		setupLog.Info("Using legacy implementation")

		if err = (&controllers.S3BucketReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "S3Bucket")
			os.Exit(1)
		}
	}

	// Setup EKSCluster Controller
	if err = (&controllers.EKSClusterReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		AWSClientFactory: awsClientFactory,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EKSCluster")
		os.Exit(1)
	}

	// Setup RouteTable Controller
	if err = (&controllers.RouteTableReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		AWSClientFactory: awsClientFactory,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RouteTable")
		os.Exit(1)
	}

	// Setup ALB Controller
	if err = (&controllers.ALBReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		AWSClientFactory: awsClientFactory,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ALB")
		os.Exit(1)
	}

	// Setup Route53HostedZone Controller
	if err = (&controllers.Route53HostedZoneReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Route53HostedZone")
		os.Exit(1)
	}

	// Setup Route53RecordSet Controller
	if err = (&controllers.Route53RecordSetReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Route53RecordSet")
		os.Exit(1)
	}

	// Setup ECSCluster Controller
	if err = (&controllers.ECSClusterReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		AWSClientFactory: awsClientFactory,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ECSCluster")
		os.Exit(1)
	}

	// Setup ElasticIP Controller
	if err = (&controllers.ElasticIPReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		AWSClientFactory: awsClientFactory,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticIP")
		os.Exit(1)
	}

	// Setup NLB Controller
	if err = (&controllers.NLBReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		AWSClientFactory: awsClientFactory,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NLB")
		os.Exit(1)
	}

	// Setup Certificate Controller
	if err = (&controllers.CertificateReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		AWSClientFactory: awsClientFactory,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Certificate")
		os.Exit(1)
	}

	// Setup APIGateway Controller
	if err = (&controllers.APIGatewayReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		AWSClientFactory: awsClientFactory,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "APIGateway")
		os.Exit(1)
	}

	// Setup CloudFront Controller
	if err = (&controllers.CloudFrontReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		AWSClientFactory: awsClientFactory,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudFront")
		os.Exit(1)
	}

	// Setup EC2KeyPair Controller
	if err = (&controllers.EC2KeyPairReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		AWSClientFactory: awsClientFactory,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EC2KeyPair")
		os.Exit(1)
	}

	// Setup ComputeStack Controller
	if err = (&controllers.ComputeStackReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		AWSClientFactory: awsClientFactory,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ComputeStack")
		os.Exit(1)
	}

	// Setup SetupEKS Controller
	if err = (&controllers.SetupEKSReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		AWSClientFactory: awsClientFactory,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SetupEKS")
		os.Exit(1)
	}

	// TODO: Add more controllers here
	// Each controller receives only the dependencies it needs:
	//
	// Lambda Controller:
	// - awsClientFactory
	// - lambdaUseCase (injected with lambdaRepository)
	//
	// DynamoDB Controller:
	// - awsClientFactory
	// - dynamoDBUseCase (injected with dynamoDBRepository)

	// Health checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// isServeCommand verifica se os argumentos indicam modo API server
func isServeCommand(args []string) bool {
	if len(args) < 2 {
		return false
	}
	return args[1] == "serve" || args[1] == "server" || args[1] == "api"
}

// runAPIServer inicia o servidor API REST
func runAPIServer() {
	// Parse flags para o servidor API
	var port int
	var host string
	var stateDir string
	var region string
	var endpoint string
	var apiKeys string
	var corsOrigins string

	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch {
		case arg == "--port" || arg == "-p":
			if i+1 < len(os.Args) {
				fmt.Sscanf(os.Args[i+1], "%d", &port)
				i++
			}
		case arg == "--host":
			if i+1 < len(os.Args) {
				host = os.Args[i+1]
				i++
			}
		case arg == "--state-dir":
			if i+1 < len(os.Args) {
				stateDir = os.Args[i+1]
				i++
			}
		case arg == "--region":
			if i+1 < len(os.Args) {
				region = os.Args[i+1]
				i++
			}
		case arg == "--endpoint":
			if i+1 < len(os.Args) {
				endpoint = os.Args[i+1]
				i++
			}
		case arg == "--api-keys":
			if i+1 < len(os.Args) {
				apiKeys = os.Args[i+1]
				i++
			}
		case arg == "--cors-origins":
			if i+1 < len(os.Args) {
				corsOrigins = os.Args[i+1]
				i++
			}
		case arg == "--help" || arg == "-h":
			printServeUsage()
			return
		}
	}

	// Defaults
	if port == 0 {
		port = 8080
	}
	if host == "" {
		host = "0.0.0.0"
	}

	// AWS config
	awsConfig := core.AWSConfig{
		Region:   region,
		Endpoint: endpoint,
	}

	// Auth config
	authConfig := api.AuthConfig{
		Enabled: apiKeys != "",
	}
	if apiKeys != "" {
		authConfig.APIKeys = strings.Split(apiKeys, ",")
	}

	// CORS origins
	var origins []string
	if corsOrigins != "" {
		origins = strings.Split(corsOrigins, ",")
	} else {
		origins = []string{"*"}
	}

	// Cria servidor
	server, err := api.NewServer(&api.ServerConfig{
		Port:           port,
		Host:           host,
		Version:        "1.0.1",
		StateDir:       stateDir,
		AllowedOrigins: origins,
		Auth:           authConfig,
		AWS:            awsConfig,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao criar servidor: %v\n", err)
		os.Exit(1)
	}

	// Inicia servidor
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao iniciar servidor: %v\n", err)
		os.Exit(1)
	}
}

// printServeUsage imprime a ajuda do comando serve
func printServeUsage() {
	fmt.Println(`
Infra Operator AWS - Modo API Server

Uso:
  infra-operator serve [flags]

Flags:
  -p, --port int             Porta do servidor (padrão: 8080)
  --host string              Host do servidor (padrão: 0.0.0.0)
  --state-dir string         Diretório de estado (padrão: ~/.infra-operator/state)
  --region string            Região AWS (padrão: us-east-1 ou env AWS_REGION)
  --endpoint string          URL do endpoint AWS (para LocalStack)
  --api-keys string          API keys separadas por vírgula (habilita autenticação)
  --cors-origins string      Origins permitidas para CORS (padrão: *)

Exemplos:
  # Inicia servidor na porta 8080
  infra-operator serve

  # Inicia com autenticação via API key
  infra-operator serve --api-keys "key1,key2"

  # Inicia com endpoint LocalStack
  infra-operator serve --endpoint http://localhost:4566

  # Inicia em porta customizada
  infra-operator serve --port 3000

Endpoints:
  GET  /health              - Health check
  POST /api/v1/plan         - Gera plano de execução
  POST /api/v1/apply        - Aplica recursos
  DELETE /api/v1/resources  - Deleta recursos
  GET  /api/v1/resources    - Lista recursos
`)
}
