/*
Copyright 2022 Dan Finneran.

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

package controllers

import (
	"context"
	helmlog "log"
	"os"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"

	vclusterv1 "github.com/thebsdbox/kubernetes-controllers/vcluster/api/v1"
	"helm.sh/helm/v3/pkg/repo"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=vcluster.fnnrn.me,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=vcluster.fnnrn.me,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=vcluster.fnnrn.me,resources=clusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// var clusters, waitForDelete int
	// var cidr, version string

	// flag.StringVar(&cidr, "cidr", "10.96.0.0/12", "CIDR RANGE")
	// flag.StringVar(&version, "version", "0.5.3", "vCluster version")

	// flag.IntVar(&clusters, "c", 10, "Number of clusters to create")
	// flag.IntVar(&waitForDelete, "wait", 0, "Time in Minutes before clusters will auto-destruct (0 wont remove them)")

	// flag.Parse()

	var cluster vclusterv1.Cluster
	if err := r.Get(ctx, req.NamespacedName, &cluster); err != nil {
		if errors.IsNotFound(err) {
			// object not found, could have been deleted after
			// reconcile request, hence don't requeue
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Directions object")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// your logic here
	log.Info("Reconciling Cluster", "Cluster", cluster.Spec.Name)

	settings := cli.New()
	settings.Debug = true

	charturl, err := repo.FindChartInRepoURL("https://charts.loft.sh", "vcluster", cluster.Spec.Version, "", "", "", getter.All(settings))
	if err != nil {
		logrus.Fatal(err)

	} else {
		logrus.Info(charturl)
	}

	chartDownloader := downloader.ChartDownloader{
		Out:     os.Stdout,
		Getters: getter.All(settings),
		Verify:  downloader.VerifyIfPossible,
	}
	filename, _, err := chartDownloader.DownloadTo(charturl, cluster.Spec.Version, settings.RepositoryCache)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Info(filename)

	namespace := "default"
	actionConfig := new(action.Configuration)
	if namespace == "" {
		namespace = settings.Namespace()
	}

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), helmlog.Printf); err != nil {
		logrus.Fatal(err)
	}

	actionClient := action.NewInstall(actionConfig)
	if actionClient.Version == "" {
		actionClient.Version = ">0.0.0-0"
	}
	actionClient.CreateNamespace = true // blah
	//client.IncludeCRDs = true
	//client.Wait = true

	chrt, err := loader.Load(filename)
	if err != nil {
		logrus.Fatal(err)
	}

	//HACK THE ~PLANET~ VALUES
	chrt.Values["serviceCIDR"] = cluster.Spec.CIDR

	chrt.Values["vcluster"].(map[string]interface{})["resources"].(map[string]interface{})["requests"].(map[string]interface{})["cpu"] = "50m"

	actionClient.Namespace = cluster.Spec.Namespace
	actionClient.ReleaseName = cluster.Spec.Name
	_, err = actionClient.Run(chrt, nil)
	if err != nil {
		logrus.Error(err)
	} else {
		cluster.Status.Status = "Provisioned"
		err = r.Client.Status().Update(context.TODO(), &cluster, &client.UpdateOptions{})
		if err != nil {
			log.Error(err, "unable to update journey")
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vclusterv1.Cluster{}).
		Complete(r)
}
