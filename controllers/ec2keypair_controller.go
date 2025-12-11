package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	keypairadapter "infra-operator/internal/adapters/aws/keypair"
	"infra-operator/internal/domain/keypair"
	keypairusecase "infra-operator/internal/usecases/keypair"
	"infra-operator/pkg/clients"
)

const ec2KeyPairFinalizer = "ec2keypair.aws-infra-operator.runner.codes/finalizer"

// EC2KeyPairReconciler reconcilia o recurso EC2KeyPair
type EC2KeyPairReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=ec2keypairs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=ec2keypairs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=ec2keypairs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile é a função principal de reconciliação do controller
func (r *EC2KeyPairReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Busca a instância EC2KeyPair
	cr := &infrav1alpha1.EC2KeyPair{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Obtém a configuração AWS
	awsCfg, _, err := r.AWSClientFactory.GetAWSConfigFromProviderRef(ctx, cr.Namespace, cr.Spec.ProviderRef)
	if err != nil {
		logger.Error(err, "Falha ao obter configuração AWS")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Cria o repositório e o caso de uso
	repo := keypairadapter.NewRepository(awsCfg)
	uc := keypairusecase.NewKeyPairUseCase(repo)

	// Mapeia o CR para o domínio
	kp := r.crToDomain(cr)

	// Trata a deleção
	if !cr.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(cr, ec2KeyPairFinalizer) {
			if err := uc.DeleteKeyPair(ctx, kp); err != nil {
				logger.Error(err, "Falha ao deletar par de chaves")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, err
			}

			controllerutil.RemoveFinalizer(cr, ec2KeyPairFinalizer)
			if err := r.Update(ctx, cr); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Adiciona finalizer
	if !controllerutil.ContainsFinalizer(cr, ec2KeyPairFinalizer) {
		controllerutil.AddFinalizer(cr, ec2KeyPairFinalizer)
		if err := r.Update(ctx, cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Sincroniza o par de chaves
	if err := uc.SyncKeyPair(ctx, kp); err != nil {
		logger.Error(err, "Falha ao sincronizar par de chaves")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Cria o secret com a chave privada se ela foi gerada agora
	if kp.PrivateKeyMaterial != "" && cr.Spec.SecretRef != nil {
		if err := r.createPrivateKeySecret(ctx, cr, kp); err != nil {
			logger.Error(err, "Falha ao criar secret da chave privada")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		cr.Status.SecretCreated = true
	}

	// Atualiza o status
	r.domainToStatus(kp, cr)

	if err := r.Status().Update(ctx, cr); err != nil {
		logger.Error(err, "Falha ao atualizar status")
		return ctrl.Result{}, err
	}

	logger.Info("EC2KeyPair reconciliado com sucesso", "keyName", kp.KeyName, "keyPairID", kp.KeyPairID)
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// crToDomain converte o CR para o modelo de domínio
func (r *EC2KeyPairReconciler) crToDomain(cr *infrav1alpha1.EC2KeyPair) *keypair.KeyPair {
	keyName := cr.Spec.KeyName
	if keyName == "" {
		keyName = cr.Name
	}

	// Garante que as tags incluem Name
	tags := cr.Spec.Tags
	if tags == nil {
		tags = make(map[string]string)
	}
	if _, exists := tags["Name"]; !exists {
		tags["Name"] = cr.Name
	}

	kp := &keypair.KeyPair{
		KeyName:           keyName,
		PublicKeyMaterial: cr.Spec.PublicKeyMaterial,
		Tags:              tags,
		DeletionPolicy:    cr.Spec.DeletionPolicy,
	}

	if cr.Status.KeyPairID != "" {
		kp.KeyPairID = cr.Status.KeyPairID
		kp.KeyFingerprint = cr.Status.KeyFingerprint
		kp.KeyType = cr.Status.KeyType
	}

	return kp
}

// domainToStatus atualiza o status do CR com dados do domínio
func (r *EC2KeyPairReconciler) domainToStatus(kp *keypair.KeyPair, cr *infrav1alpha1.EC2KeyPair) {
	now := metav1.Now()
	cr.Status.Ready = kp.IsReady()
	cr.Status.KeyPairID = kp.KeyPairID
	cr.Status.KeyFingerprint = kp.KeyFingerprint
	cr.Status.KeyName = kp.KeyName
	cr.Status.KeyType = kp.KeyType
	cr.Status.LastSyncTime = &now
}

// createPrivateKeySecret cria um secret Kubernetes com a chave privada
func (r *EC2KeyPairReconciler) createPrivateKeySecret(ctx context.Context, cr *infrav1alpha1.EC2KeyPair, kp *keypair.KeyPair) error {
	if cr.Spec.SecretRef == nil {
		return nil
	}
	secretNamespace := cr.Spec.SecretRef.Namespace
	if secretNamespace == "" {
		secretNamespace = cr.Namespace
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Spec.SecretRef.Name,
			Namespace: secretNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":  "infra-operator",
				"aws-infra-operator.runner.codes/keypair": cr.Name,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"private-key": []byte(kp.PrivateKeyMaterial),
			"key-name":    []byte(kp.KeyName),
		},
	}

	// Define owner reference se estiver no mesmo namespace
	if secretNamespace == cr.Namespace {
		if err := controllerutil.SetControllerReference(cr, secret, r.Scheme); err != nil {
			return fmt.Errorf("falha ao definir owner reference: %w", err)
		}
	}

	// Cria ou atualiza o secret
	existingSecret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Name: secret.Name, Namespace: secret.Namespace}, existingSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, secret)
		}
		return err
	}

	// Secret já existe, não sobrescreve (a chave privada só deve ser armazenada uma vez)
	return nil
}

// SetupWithManager configura o controller com o manager
func (r *EC2KeyPairReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.EC2KeyPair{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
