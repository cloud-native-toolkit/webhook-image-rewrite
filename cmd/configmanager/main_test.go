package configmanager

import (
	"github.com/ibm-garage-cloud/webhook-image-rewrite/cmd/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func buildConfigManager() ConfigManager {
	imageMappings := []model.ImageMapping{
		model.ImageMapping{Source: "source.co", Mirror: "mirror.co"},
	}
	ignoredNamespaces := []string{"ns1", "ns2"}

	config := model.Config{
		DefaultHost: "docker.io",
		ImageMappings: imageMappings,
		IgnoredNamespaces: ignoredNamespaces,
	}

	return ConfigManager{Config: &config}
}

func buildPod(images ...string) *corev1.Pod {
	containers := make([]corev1.Container, 0)

	for _, image := range images {
		container := corev1.Container{Image: image}
		containers = append(containers, container)
	}

	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{GenerateName: "test"}, Spec: corev1.PodSpec{Containers: containers}}
}

func TestConfigManager_MutationRequired_ignoredNamespaces(t *testing.T) {
	configmanager := buildConfigManager()

	pod := buildPod("source.co/test")

	if configmanager.MutationRequired("ns1", pod) {
		t.Error("Should skip mutation for namespace")
	}
}

func TestConfigManager_MutationRequired_allowedNamespace(t *testing.T) {
	configmanager := buildConfigManager()

	pod := buildPod("source.co/test")

	if !configmanager.MutationRequired("test", pod) {
		t.Error("Should not skip mutation for namespace")
	}
}

func TestConfigManager_patchImages(t *testing.T) {
	configmanager := buildConfigManager()

	pod := buildPod("skip.co/test1", "source.co/test2")

	result := configmanager.patchImages(pod)

	if result[0].Op != "replace" {
		t.Error("Operation should be replace")
	}

	if result[0].Value != "mirror.co/test2" {
		t.Error("Value should be mirror.co/test2")
	}

	if result[0].Path != "/spec/containers/1/image" {
		t.Error("Path should be /spec/containers/1/image")
	}
}