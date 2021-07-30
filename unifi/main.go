/*
Copyright 2021 Dan Finneran.

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
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	restClient "k8s.io/client-go/rest"
	cmdClient "k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/paultyng/go-unifi/unifi"

	unifiv1 "github.com/thebsdbox/kubernetes-controllers/unifi/api/v1"
	"github.com/thebsdbox/kubernetes-controllers/unifi/controllers"
	unifiReconciler "github.com/thebsdbox/kubernetes-controllers/unifi/pkg/unifi"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(unifiv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8082", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "unifi-controller",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	u, err := CloudControllerLogin()
	if err != nil {
		setupLog.Error(err, "unable to connect to Cloud Controller")
		os.Exit(1)
	}

	// Start the Unifi Reconciler
	go unifiReconciler.Reconcile(logger, u, mgr.GetClient())

	if err = (&controllers.CloudControllerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr, u); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudController")
		os.Exit(1)
	}
	if err = (&controllers.UserReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "User")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

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

func CloudControllerLogin() (*unifi.Client, error) {
	// So we have a Kubernetes cluster in r.Client, however we can't use it until the caches
	// start otherwise it will just return an error. So in order to get things ready we will
	// use our own client in order to get the key and set up the Google Maps client in advance
	config, err := restClient.InClusterConfig()
	if err != nil {
		kubeConfig := cmdClient.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
		config, err = cmdClient.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			return nil, err
		}
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	secret, err := clientset.CoreV1().Secrets("default").Get(context.TODO(), "unifi", v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	user := string(secret.Data["user"])
	if user == "" {
		return nil, fmt.Errorf("no Unifi Username found within secret")
	}

	pass := string(secret.Data["pass"])
	if pass == "" {
		return nil, fmt.Errorf("no Unifi Password found within secret")
	}

	baseURL := string(secret.Data["url"])
	if baseURL == "" {
		return nil, fmt.Errorf("no Unifi URL found within secret")
	}

	// u, err := url.Parse(baseURL)
	// if err != nil {
	// 	return nil, err
	// }

	// if u.Port() == "" {
	// }

	uClient := &unifi.Client{}

	err = uClient.SetBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{}
	httpClient.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	jar, _ := cookiejar.New(nil)
	httpClient.Jar = jar

	uClient.SetHTTPClient(httpClient)
	err = uClient.Login(context.TODO(), user, pass)
	if err != nil {
		return nil, err
	}

	return uClient, nil
}
