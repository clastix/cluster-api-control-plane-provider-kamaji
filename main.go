// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"os"

	kamajiv1alpha1 "github.com/clastix/kamaji/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/component-base/featuregate"
	capiv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util/flags"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	controlplanev1alpha1 "github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha1"
	"github.com/clastix/cluster-api-control-plane-provider-kamaji/controllers"
	"github.com/clastix/cluster-api-control-plane-provider-kamaji/pkg/externalclusterreference"
	"github.com/clastix/cluster-api-control-plane-provider-kamaji/pkg/features"
	"github.com/clastix/cluster-api-control-plane-provider-kamaji/pkg/indexers"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kamajiv1alpha1.AddToScheme(scheme))
	utilruntime.Must(capiv1beta2.AddToScheme(scheme))

	utilruntime.Must(controlplanev1alpha1.AddToScheme(scheme))
}

//nolint:funlen,cyclop
func main() {
	var dynamicInfraClusters []string

	metricsAddr, enableLeaderElection, probeAddr, maxConcurrentReconciles, managerOpts := "", false, "", 1, flags.ManagerOptions{}

	flagSet := pflag.CommandLine

	flagSet.StringSliceVar(&dynamicInfraClusters, "dynamic-infrastructure-clusters", nil, "When the DynamicInfrastructureClusterPatch feature flag is enabled, "+
		"allows specifying which Infrastructure Clusters can be dynamically patched. "+
		"This feature is useful for developers of custom or non public Cluster API infrastructure providers.")
	flagSet.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flagSet.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flagSet.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flagSet.IntVar(&maxConcurrentReconciles, "max-concurrent-reconciles", 1, "The maximum number of concurrent KamajiControlPlane reconciles which can be run")
	// zap logging FlagSet
	var goFlagSet flag.FlagSet

	opts := zap.Options{Development: true}
	opts.BindFlags(&goFlagSet)

	flags.AddManagerOptions(flagSet, &managerOpts)
	flagSet.AddGoFlagSet(&goFlagSet)

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	featureGate := featuregate.NewFeatureGate()

	if err := featureGate.Add(map[featuregate.Feature]featuregate.FeatureSpec{
		features.ExternalClusterReference: {
			Default:       false,
			LockToDefault: false,
			PreRelease:    featuregate.Alpha,
		},
		features.ExternalClusterReferenceCrossNamespace: {
			Default:       false,
			LockToDefault: false,
			PreRelease:    featuregate.Alpha,
		},
		features.SkipInfraClusterPatch: {
			Default:       false,
			LockToDefault: false,
			PreRelease:    featuregate.Alpha,
		},
		features.DynamicInfrastructureClusterPatch: {
			Default:       false,
			LockToDefault: false,
			PreRelease:    featuregate.Alpha,
		},
	}); err != nil {
		setupLog.Error(err, "unable to add feature gates")
		os.Exit(1)
	}

	featureGate.AddFlag(flagSet)

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		setupLog.Error(err, "unable to parse arguments")
		os.Exit(1)
	}

	_, metricsOpts, err := flags.GetManagerOptions(managerOpts)
	if err != nil {
		setupLog.Error(err, "Unable to start manager: invalid flags")
		os.Exit(1)
	}

	if !featureGate.Enabled(features.DynamicInfrastructureClusterPatch) && len(dynamicInfraClusters) > 0 {
		setupLog.Error(errors.New("cannot set dynamic infrastructure clusters when the feature flag is disabled"), "DynamicInfrastructureClusterPatch feature flag is disabled")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:  scheme,
		Metrics: *metricsOpts,
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: 9443, //nolint:mnd
		}),
		NewClient: func(config *rest.Config, options client.Options) (client.Client, error) {
			options.Cache.Unstructured = true

			return client.New(config, options)
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "kamaji.controlplane.cluster.x-k8s.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ecrStore, triggerChannel := externalclusterreference.NewStore(), make(chan event.GenericEvent)

	if err = (&controllers.KamajiControlPlaneReconciler{
		ExternalClusterReferenceStore: ecrStore,
		FeatureGates:                  featureGate,
		MaxConcurrentReconciles:       maxConcurrentReconciles,
		DynamicInfrastructureClusters: sets.New[string](dynamicInfraClusters...),
	}).SetupWithManager(ctx, mgr, triggerChannel); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KamajiControlPlane")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if featureGate.Enabled(features.ExternalClusterReference) || featureGate.Enabled(features.ExternalClusterReferenceCrossNamespace) {
		if err = indexers.SetupWithManager(ctx, mgr); err != nil {
			setupLog.Error(err, "unable to create indexers")
			os.Exit(1)
		}

		if err = (&controllers.ExternalClusterReferenceReconciler{Client: mgr.GetClient(), Store: ecrStore, TriggerChannel: triggerChannel}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ExternalClusterReference")
			os.Exit(1)
		}
	}

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")

	if err = mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
