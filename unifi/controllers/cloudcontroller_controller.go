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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/paultyng/go-unifi/unifi"
	unifiv1 "github.com/thebsdbox/kubernetes-controllers/unifi/api/v1"
)

// CloudControllerReconciler reconciles a CloudController object
type CloudControllerReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	uClient *unifi.Client
}

//+kubebuilder:rbac:groups=unifi.thebsdbox.co.uk,resources=cloudcontrollers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=unifi.thebsdbox.co.uk,resources=cloudcontrollers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=unifi.thebsdbox.co.uk,resources=cloudcontrollers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CloudController object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CloudControllerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	fmt.Println("Cloud Controller Reconcing !")
	log := log.FromContext(ctx)

	sites, err := r.uClient.ListSites(context.TODO())
	if err != nil {
		return ctrl.Result{}, err
	}
	for x := range sites {
		log.Info("Sites", "Name", sites[x].Name, "ID", sites[x].ID)
	}
	// your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudControllerReconciler) SetupWithManager(mgr ctrl.Manager, u *unifi.Client) error {
	r.uClient = u

	return ctrl.NewControllerManagedBy(mgr).
		For(&unifiv1.CloudController{}).
		Complete(r)
}
