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

package controllers

import (
	"context"
	"fmt"

	strip "github.com/grokify/html-strip-tags-go"
	katnavv1 "github.com/thebsdbox/kubernetes-controllers/katnav/api/v1"
	"googlemaps.github.io/maps"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	restClient "k8s.io/client-go/rest"
	cmdClient "k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DirectionsReconciler reconciles a Directions object
type DirectionsReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	mClient *maps.Client
}

//+kubebuilder:rbac:groups=katnav.fnnrn.me,resources=directions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=katnav.fnnrn.me,resources=directions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=katnav.fnnrn.me,resources=directions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Directions object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *DirectionsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Set the log to an acutal value so we can create log messages
	log := log.FromContext(ctx)
	log.Info("Reconciling direction resources")

	var directions katnavv1.Directions
	if err := r.Get(ctx, req.NamespacedName, &directions); err != nil {
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
	log.Info("Determining journey", "Source", directions.Spec.Source, "Destination", directions.Spec.Destination)

	request := &maps.DirectionsRequest{
		Origin:      directions.Spec.Source,
		Destination: directions.Spec.Destination,
		Mode:        maps.TravelModeDriving,
	}

	route, _, err := r.mClient.Directions(context.Background(), request)
	if err != nil {
		log.Error(err, "unable to fetch Directions")
	}
	var directionsString string
	if len(route) == 0 {
		return ctrl.Result{}, nil
	}

	log.Info("New Route", "Summary", route[0].Summary)
	for x := range route[0].Legs {
		for y := range route[0].Legs[x].Steps {
			stripped := strip.StripTags(route[0].Legs[x].Steps[y].HTMLInstructions)
			directionsString += stripped + "\n"
		}
		directions.Status.Distance = route[0].Legs[x].Distance.HumanReadable
		directions.Status.Duration = fmt.Sprintf("Total Minutes: %f", route[0].Legs[x].Duration.Minutes())
		directions.Status.StartLocation = route[0].Legs[x].StartAddress
		directions.Status.EndLocation = route[0].Legs[x].EndAddress

	}

	directions.Status.RouteSummary = route[0].Summary
	directions.Status.Directions = directionsString

	err = r.Client.Status().Update(context.TODO(), &directions, &client.UpdateOptions{})
	if err != nil {
		log.Error(err, "unable to update journey")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DirectionsReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// So we have a Kubernetes cluster in r.Client, however we can't use it until the caches
	// start otherwise it will just return an error. So in order to get things ready we will
	// use our own client in order to get the key and set up the Google Maps client in advance
	config, err := restClient.InClusterConfig()
	if err != nil {
		kubeConfig :=
			cmdClient.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
		config, err = cmdClient.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			return err
		}
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	secret, err := clientset.CoreV1().Secrets("default").Get(context.TODO(), "katnav", v1.GetOptions{})
	if err != nil {
		return err
	}

	token := string(secret.Data["directionsKey"])
	if token == "" {
		return fmt.Errorf("no Token found within API Key")
	}
	r.mClient, err = maps.NewClient(maps.WithAPIKey(token))
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&katnavv1.Directions{}).
		Complete(r)
}
