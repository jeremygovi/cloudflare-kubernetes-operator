/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"
	"time"

	"github.com/cloudflare/cloudflare-go"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	cloudflarev1 "github.com/jeremygovi/cloudflare-kubernetes-operator/api/v1"
	"github.com/jeremygovi/cloudflare-kubernetes-operator/internal/controller"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(cloudflarev1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var requeueDuration time.Duration

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.DurationVar(&requeueDuration, "requeue-duration", 5*time.Minute,
		"Duration after which resources are requeued for reconciliation.")

	opts := zap.Options{
		Development: false, // Set to false to respect log level flags
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Try in-cluster config first, then fallback to kubeconfig
	config, err := ctrl.GetConfig()
	if err != nil {
		setupLog.Error(err, "unable to get kubeconfig")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: 9443,
		}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "cloudflare-operator-lock",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Initialize Cloudflare API client
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	if apiToken == "" {
		setupLog.Error(nil, "CLOUDFLARE_API_TOKEN environment variable is required")
		os.Exit(1)
	}

	cloudflareAPI, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		setupLog.Error(err, "unable to create Cloudflare API client")
		os.Exit(1)
	}

	// Optional: Log account ID if provided
	if accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID"); accountID != "" {
		setupLog.Info("Using Cloudflare Account ID", "accountID", accountID[:8]+"...")
	}

	// Setup CloudflareRecord controller
	if err = (&controller.CloudflareRecordReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		CloudflareAPI:   cloudflareAPI,
		RequeueDuration: requeueDuration,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudflareRecord")
		os.Exit(1)
	}

	// Setup CloudflareRuleset controller
	if err = (&controller.CloudflareRulesetReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		CloudflareAPI:   cloudflareAPI,
		RequeueDuration: requeueDuration,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudflareRuleset")
		os.Exit(1)
	}

	// Setup CloudflareZone controller
	if err = (&controller.CloudflareZoneReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		CloudflareAPI:   cloudflareAPI,
		RequeueDuration: requeueDuration,
		AccountID:       os.Getenv("CLOUDFLARE_ACCOUNT_ID"), // Optional: default account ID
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudflareZone")
		os.Exit(1)
	}

	// Add health and readiness checks
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
