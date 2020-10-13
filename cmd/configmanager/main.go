package configmanager

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/ibm-garage-cloud/webhook-image-rewrite/cmd/model"
	"github.com/ibm-garage-cloud/webhook-image-rewrite/util"
	corev1 "k8s.io/api/core/v1"
	"reflect"
	"strings"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type ConfigManager struct {
	Config *model.Config
}

func (c ConfigManager) MutationRequired(namespace string, pod *corev1.Pod) bool {
	matchesCurrentNamespace := func (value interface{}) bool {
		return value == namespace
	}

	if util.Any(c.Config.IgnoredNamespaces, matchesCurrentNamespace) {
		glog.Infof("Skip mutation for %v (%v) for it's in special namespace: %v", pod.Name, pod.GenerateName, namespace)
		return false
	}

	mapContainerImage := func (c interface{}) interface{} {
		return c.(corev1.Container).Image
	}

	images := util.Map(pod.Spec.Containers, mapContainerImage)
	required := util.Any(images, c.matchMappingSource)

	glog.Infof("Mutation policy for %v/%v (%v) required: %v", namespace, pod.Name, pod.GenerateName, required)

	return required
}

func (c ConfigManager) matchMappingSource(val interface{}) bool {
	image := val.(string)

	sourceImage := *getSourceImage(&image, c.Config.DefaultHost)

	return util.Any(c.Config.ImageMappings, hasSourcePrefix(sourceImage))
}

func hasSourcePrefix(sourceImage string) func (val interface{}) bool {
	return func (val interface{}) bool {
		mapping := val.(model.ImageMapping)

		return strings.HasPrefix(sourceImage, mapping.Source)
	}
}

func (c ConfigManager) rewriteImage(image *string) *string {
	config := *c.Config

	sourceImage := *getSourceImage(image, config.DefaultHost)

	f := util.Filter(config.ImageMappings, hasSourcePrefix(sourceImage))
	filteredImageMappings := reflect.ValueOf(f)

	if filteredImageMappings.Len() == 0 {
		return image
	}

	mapping := filteredImageMappings.Index(0).Interface().(model.ImageMapping)

	newImage := strings.ReplaceAll(sourceImage, mapping.Source, mapping.Mirror)

	return &newImage
}

// create mutation patch for resoures
func (c ConfigManager) CreatePatch(pod *corev1.Pod) ([]byte, error) {
	var patch []patchOperation

	patch = append(patch, c.patchImages(pod)...)

	return json.Marshal(patch)
}

func (c ConfigManager) patchImages(pod *corev1.Pod) []patchOperation {
	var result []patchOperation
	result = make([]patchOperation, 0)

	for index, container := range pod.Spec.Containers {
		patch := c.patchImage(index, &container.Image)

		// Only apply the patch if we are changing the value
		if patch.Value != container.Image {
			result = append(result, patch)
		}
	}

	return result
}

func (c ConfigManager) patchImage(containerIndex int, image *string) patchOperation {
	newImage := c.rewriteImage(image)

	return patchOperation{Op: "replace", Path: fmt.Sprintf("/spec/containers/%d/image", containerIndex), Value: *newImage}
}

func getSourceImage(image *string, defaultHost string) *string {
	imageParts := strings.Split(*image, "/")

	host := defaultHost
	imageRepo := *image

	if len(imageParts) > 1 && strings.Contains(imageParts[0], ".") {
		host = imageParts[0]

		imageRepo = strings.Join(imageParts[1:], "/")
	}

	sourceImage := host + "/" + imageRepo

	return &sourceImage
}
