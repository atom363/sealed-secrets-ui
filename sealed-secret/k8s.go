package sealedsecret

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/rs/zerolog/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/client-go/util/homedir"
)

func getLocalClient() (*kubernetes.Clientset, error) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	return kubernetes.NewForConfig(config)
}

func getClusterClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	return kubernetes.NewForConfig(config)
}

func decodeSecret(secretData map[string][]byte) map[string]string {
	data := make(map[string]string)
	for key, value := range secretData {
		data[key] = string(value)
	}

	return data
}

func (s SealedSecretService) getSecretData(ctx context.Context, namespace, secretName string) (map[string]string, error) {
	secret, err := s.k8sClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		log.Warn().Msg("Secret not found")
		return nil, nil
	}

	data := decodeSecret(secret.Data)

	return data, nil
}

func (s SealedSecretService) listNamespaces(ctx context.Context) ([]string, error) {
	namespaces, err := s.k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	results := make([]string, 0, len(namespaces.Items))
	for _, namespace := range namespaces.Items {
		results = append(results, namespace.Name)
	}

	sort.Strings(results)
	return results, nil
}

func (s SealedSecretService) listSecretNames(ctx context.Context, namespace string) ([]string, error) {
	if namespace == "" {
		return []string{}, nil
	}

	secrets, err := s.k8sClient.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if apierrors.IsNotFound(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	results := make([]string, 0, len(secrets.Items))
	for _, secret := range secrets.Items {
		results = append(results, secret.Name)
	}

	sort.Strings(results)
	return results, nil
}
