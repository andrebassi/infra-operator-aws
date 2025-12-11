package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/controllers"
	"infra-operator/pkg/clients"
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
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

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
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: 9443,
		}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "infra-operator.aws-infra-operator.runner.codes",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Create AWS Client Factory
	awsClientFactory := clients.NewAWSClientFactory(mgr.GetClient())

	// Setup AWSProvider Controller
	if err = (&controllers.AWSProviderReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AWSProvider")
		os.Exit(1)
	}

	// Setup S3Bucket Controller (legacy implementation without AWSClientFactory)
	if err = (&controllers.S3BucketReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "S3Bucket")
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
