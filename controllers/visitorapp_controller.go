/*
Copyright 2023.

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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	defaultv1alpha1 "github.com/Crisarias/visitors-operator/api/v1alpha1"
)

var log = logf.Log.WithName("controller_visitorsapp")

// VisitorAppReconciler reconciles a VisitorApp object
type VisitorAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=default.example.com,resources=visitorapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=default.example.com,resources=visitorapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=default.example.com,resources=visitorapps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VisitorApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *VisitorAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling VisitorsApp")

	// TODO(user): your logic here
	// Fetch the VisitorsApp instance
	visitorApp := &defaultv1alpha1.VisitorApp{}
	err := r.Get(ctx, req.NamespacedName, visitorApp)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			reqLogger.Info("visitorApp resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get visitorApp")
		return ctrl.Result{}, err
	}

	// Check if the visitorApp instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isVisitorAppMarkedToBeDeleted := visitorApp.GetDeletionTimestamp() != nil
	if isVisitorAppMarkedToBeDeleted {
		reqLogger.Info("visitorApp must be deleted")
	}

	var result *ctrl.Result

	// == MySQL ==========
	result, err = r.ensureSecret(req, visitorApp, r.mysqlAuthSecret(visitorApp))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureDeployment(req, visitorApp, r.mysqlDeployment(visitorApp))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureService(req, visitorApp, r.mysqlService(visitorApp))
	if result != nil {
		return *result, err
	}

	mysqlRunning := r.isMysqlUp(visitorApp)

	if !mysqlRunning {
		// If MySQL isn't running yet, requeue the reconcile
		// to run again after a delay
		delay := time.Second * time.Duration(5)

		log.Info(fmt.Sprintf("MySQL isn't running, waiting for %s", delay))
		return ctrl.Result{RequeueAfter: delay}, nil
	}

	// == Visitors Backend  ==========
	result, err = r.ensureDeployment(req, visitorApp, r.backendDeployment(visitorApp))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureService(req, visitorApp, r.backendService(visitorApp))
	if result != nil {
		return *result, err
	}

	err = r.updateBackendStatus(visitorApp)
	if err != nil {
		// Requeue the request if the status could not be updated
		return ctrl.Result{}, err
	}

	result, err = r.handleBackendChanges(visitorApp)
	if result != nil {
		return *result, err
	}

	// == Visitors Frontend ==========
	result, err = r.ensureDeployment(req, visitorApp, r.frontendDeployment(visitorApp))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureService(req, visitorApp, r.frontendService(visitorApp))
	if result != nil {
		return *result, err
	}

	err = r.updateFrontendStatus(visitorApp)
	if err != nil {
		// Requeue the request
		return ctrl.Result{}, err
	}

	result, err = r.handleFrontendChanges(visitorApp)
	if result != nil {
		return *result, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VisitorAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&defaultv1alpha1.VisitorApp{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
