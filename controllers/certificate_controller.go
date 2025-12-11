package controllers

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/pkg/clients"
	"infra-operator/pkg/mapper"
)

const certificateFinalizerName = "certificate.infra.operator.aws.io/finalizer"

type CertificateReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=certificates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=certificates/finalizers,verbs=update

func (r *CertificateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	certCR := &infrav1alpha1.Certificate{}
	if err := r.Get(ctx, req.NamespacedName, certCR); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	certUseCase, err := r.AWSClientFactory.GetACMUseCase(ctx, certCR.Spec.ProviderRef, certCR.Namespace)
	if err != nil {
		logger.Error(err, "failed to get ACM use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	if !certCR.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(certCR, certificateFinalizerName) {
			cert := mapper.CRToDomainACM(certCR)
			if err := certUseCase.DeleteCertificate(ctx, cert); err != nil {
				logger.Error(err, "failed to delete certificate")
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(certCR, certificateFinalizerName)
			if err := r.Update(ctx, certCR); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(certCR, certificateFinalizerName) {
		controllerutil.AddFinalizer(certCR, certificateFinalizerName)
		if err := r.Update(ctx, certCR); err != nil {
			return ctrl.Result{}, err
		}
	}

	cert := mapper.CRToDomainACM(certCR)
	if err := certUseCase.SyncCertificate(ctx, cert); err != nil {
		logger.Error(err, "failed to sync certificate")
		certCR.Status.Ready = false
		r.Status().Update(ctx, certCR)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	mapper.DomainToStatusACM(cert, certCR)
	if err := r.Status().Update(ctx, certCR); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *CertificateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.Certificate{}).
		Complete(r)
}
