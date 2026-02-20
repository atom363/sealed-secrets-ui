package sealedsecret

import (
	"context"
	"crypto/rsa"
	"fmt"
	"strings"

	"github.com/atom363/sealed-secrets-ui/model"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type SealedSecretService struct {
	sealedSecretControllerName      string
	sealedSecretControllerNamespace string
	clusterDomain                   string
	k8sClient                       *kubernetes.Clientset
	dynamicClient                   dynamic.Interface
	annotationsToPreserve           map[string]struct{}
}

type encryptRequest struct {
	pubKey     *rsa.PublicKey
	secretName string
	namespace  string
	scope      string
	values     map[string]string
}

func NewSealedSecretService(controllerNamespace, controllerName, clusterDomain string, annotationAllowlist []string) (SealedSecretService, error) {
	config, err := getClusterConfig()
	if err != nil {
		config, err = getLocalConfig()
	}
	if err != nil {
		return SealedSecretService{}, fmt.Errorf("failed to get Kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return SealedSecretService{}, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return SealedSecretService{}, fmt.Errorf("failed to create Kubernetes dynamic client: %w", err)
	}

	return SealedSecretService{
		sealedSecretControllerNamespace: controllerNamespace,
		sealedSecretControllerName:      controllerName,
		clusterDomain:                   clusterDomain,
		k8sClient:                       clientset,
		dynamicClient:                   dynamicClient,
		annotationsToPreserve:           toStringSet(annotationAllowlist),
	}, nil
}

func (s SealedSecretService) CreateSealedSecret(ctx context.Context, opts model.CreateOpts) (string, error) {
	existingData, err := s.getSecretData(ctx, opts.Namespace, opts.SecretName)
	if err != nil {
		return "", fmt.Errorf("failed to get existing secret data: %w", err)
	}

	// we need to encrypt all the existing data as well as the new data
	valuesToEncrypt := make(map[string]string)
	for key, value := range existingData {
		valuesToEncrypt[key] = value
	}

	for key, value := range opts.Values {
		valuesToEncrypt[key] = value
	}

	// we need to get the public key every time we create a sealed secret because the
	// sealed-secrets controller rotates the public key every X hours
	pubKey, err := s.getPublicKey(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %w", err)
	}

	req := encryptRequest{
		pubKey:     pubKey,
		secretName: opts.SecretName,
		namespace:  opts.Namespace,
		values:     valuesToEncrypt,
		scope:      opts.Scope,
	}

	encryptedData, err := s.encryptValues(req)
	if err != nil {
		return "", err
	}

	scopeAnnotations := getScopeAnnotations(req.scope)
	sealedSecretAnnotations := copyStringMap(scopeAnnotations)

	preservedAnnotations, err := s.getSealedSecretAnnotations(ctx, opts.Namespace, opts.SecretName)
	if err != nil {
		return "", fmt.Errorf("failed to get existing sealed-secret annotations: %w", err)
	}

	for key, value := range preservedAnnotations {
		if isScopeAnnotation(key) {
			continue
		}
		sealedSecretAnnotations[key] = value
	}

	sealedSecret := model.SealedSecret{
		APIVersion: "bitnami.com/v1alpha1",
		Kind:       "SealedSecret",
		Metadata: model.Metadata{
			Name:        req.secretName,
			Namespace:   req.namespace,
			Annotations: sealedSecretAnnotations,
		},
		Spec: model.SealedSecretSpec{
			EncryptedData: encryptedData,
			Template: model.Template{
				Metadata: model.Metadata{
					Name:        req.secretName,
					Namespace:   req.namespace,
					Annotations: scopeAnnotations,
				},
			},
		},
	}

	yamlData, err := yaml.Marshal(sealedSecret)
	if err != nil {
		return "", fmt.Errorf("failed to marshal sealed secret to YAML: %w", err)
	}

	return string(yamlData), nil
}

func (s SealedSecretService) ListNamespaces(ctx context.Context) ([]string, error) {
	return s.listNamespaces(ctx)
}

func (s SealedSecretService) ListSecretNames(ctx context.Context, namespace string) ([]string, error) {
	return s.listSecretNames(ctx, namespace)
}

func (s SealedSecretService) getLabel(req encryptRequest) string {
	switch req.scope {
	case "cluster":
		return ""
	case "namespace":
		return req.namespace
	default:
		return fmt.Sprintf("%s/%s", req.namespace, req.secretName)
	}
}

func (s SealedSecretService) encryptValues(req encryptRequest) (map[string]string, error) {
	encryptedData := make(map[string]string)
	for key, value := range req.values {
		enc, err := hybridEncrypt(req.pubKey, value, s.getLabel(req))
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt value: %w", err)
		}

		encryptedData[key] = enc
	}

	return encryptedData, nil
}

func toStringSet(values []string) map[string]struct{} {
	results := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		results[value] = struct{}{}
	}

	return results
}

func copyStringMap(source map[string]string) map[string]string {
	results := make(map[string]string, len(source))
	for key, value := range source {
		results[key] = value
	}

	return results
}

func getScopeAnnotations(scope string) map[string]string {
	switch scope {
	case "cluster":
		return map[string]string{"sealedsecrets.bitnami.com/cluster-wide": "true"}
	case "namespace":
		return map[string]string{"sealedsecrets.bitnami.com/namespace-wide": "true"}
	default:
		return map[string]string{}
	}
}

func isScopeAnnotation(annotationKey string) bool {
	switch annotationKey {
	case "sealedsecrets.bitnami.com/cluster-wide", "sealedsecrets.bitnami.com/namespace-wide":
		return true
	default:
		return false
	}
}
