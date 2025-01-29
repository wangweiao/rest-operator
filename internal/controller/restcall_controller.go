/*
Copyright 2025.

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

package controller

import (
	"context"
	"io"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	examplev1alpha1 "github.com/example/rest-operator/api/v1alpha1"
)

// RestCallReconciler reconciles a RestCall object
type RestCallReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	HTTPClient *http.Client
}

// +kubebuilder:rbac:groups=example.example.com,resources=restcalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=example.example.com,resources=restcalls/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=example.example.com,resources=restcalls/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RestCall object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *RestCallReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var restCall examplev1alpha1.RestCall
	if err := r.Get(ctx, req.NamespacedName, &restCall); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	endpoint := restCall.Spec.Endpoint
	headers := restCall.Spec.Headers

	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		logger.Error(err, "Failed to create request")
		return ctrl.Result{}, err
	}

	for k, v := range headers {
		httpReq.Header.Add(k, v)
	}

	client := r.HTTPClient
	if client == nil {
		client = &http.Client{}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Error(err, "Failed to make HTTP call")
		return ctrl.Result{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err, "Failed to read response body")
		return ctrl.Result{}, err
	}

	responseStr := string(body)
	logger.Info("REST call response", "response", responseStr)

	restCall.Status.LastCallTime = time.Now().Format(time.RFC3339)
	restCall.Status.Response = responseStr

	if err := r.Status().Update(ctx, &restCall); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RestCallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplev1alpha1.RestCall{}).
		Complete(r)
}
