package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/kubernetes/pkg/apis/core/v1"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

var ignoredNamespaces = []string{
	metav1.NamespaceSystem,
	metav1.NamespacePublic,
	"image-rewrite",
}

const (
	admissionWebhookAnnotationInjectKey = "image-rewrite.toolkit.ibm/inject"
	admissionWebhookAnnotationStatusKey = "image-rewrite.toolkit.ibm/status"
)

type WebhookServer struct {
	rewriteConfig *Config
	server        *http.Server
}

// Webhook Server parameters
type WhSvrParameters struct {
	port           int    // webhook server port
	certFile       string // path to the x509 certificate for https
	keyFile        string // path to the x509 private key matching `CertFile`
	rewriteCfgFile string // path to sidecar injector configuration file
}

type Config struct {
	DefaultHost string `yaml:"defaultHost"`
	ImageMappings []ImageMapping `yaml:"imageMappings"`
}

// constructor function
func(c *Config) fill_defaults(){

	// setting default values
	// if no values present
	if c.DefaultHost == "" {
		c.DefaultHost = "docker.io"
	}
}

type ImageMapping struct {
	Source string `yaml:"source"`
	Mirror  string `yaml:"mirror"`
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	// defaulting with webhooks:
	// https://github.com/kubernetes/kubernetes/issues/57982
	_ = v1.AddToScheme(runtimeScheme)
}

// (https://github.com/kubernetes/kubernetes/issues/57982)
func applyDefaultsWorkaround(containers []corev1.Container, volumes []corev1.Volume) {
	defaulter.Default(&corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: containers,
			Volumes:    volumes,
		},
	})
}

func loadConfig(configFile string) (*Config, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	glog.Infof("New configuration: sha256sum %x", sha256.Sum256(data))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Check whether the target resource needs to be mutated
func mutationRequired(req *v1beta1.AdmissionRequest,ignoredList *[]string, pod *corev1.Pod, config *Config) bool {

	// skip special kubernetes system namespaces
	for _, namespace := range *ignoredList {
		if req.Namespace == namespace {
			glog.Infof("Skip mutation for %v (%v) for it's in special namespace: %v", pod.Name, pod.GenerateName, req.Namespace)
			return false
		}
	}

	images := mapContainerImages(&pod.Spec.Containers)

	required := doesHostMatch(images, config)

	glog.Infof("Mutation policy for %v/%v (%v) required: %v", req.Namespace, pod.Name, pod.GenerateName, required)

	return required
}

func mapContainerImages(c *[]corev1.Container) *[]string {
	containers := *c

	var images []string
	images = make([]string, 0)

	for _, container := range containers {
		images = append(images, container.Image)
	}

	return &images
}

func doesHostMatch(images *[]string, config *Config) bool {
	val := *images

	for _, image := range val {
		newImage := *rewriteImage(&image, config)

		if newImage != image {
			return true
		}
	}

	return false
}

func patchImages(pod *corev1.Pod, rewriteConfig *Config) []patchOperation {
	var result []patchOperation
	result = make([]patchOperation, 0)

	for index, container := range pod.Spec.Containers {
		result = append(result, patchImage(index, &container.Image, rewriteConfig))
	}

	return result
}

func patchImage(containerIndex int, image *string, config *Config) patchOperation {
	newImage := rewriteImage(image, config)

	return patchOperation{Op: "replace", Path: fmt.Sprintf("/spec/containers/%d/image", containerIndex), Value: *newImage}
}

func rewriteImage(image *string, rewriteConfig *Config) *string {
	config := *rewriteConfig

	sourceImage := *getSourceImage(image, config.DefaultHost)

	for _, mapping := range config.ImageMappings {
		if strings.HasPrefix(sourceImage, mapping.Source) {
			newImage := strings.ReplaceAll(sourceImage, mapping.Source, mapping.Mirror)

			return &newImage
		}
	}

	return image
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

// create mutation patch for resoures
func createPatch(pod *corev1.Pod, rewriteConfig *Config, annotations map[string]string) ([]byte, error) {
	var patch []patchOperation

	patch = append(patch, patchImages(pod, rewriteConfig)...)

	return json.Marshal(patch)
}

// main mutation process
func (whsvr *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		glog.Errorf("Could not unmarshal raw object: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, pod.GenerateName, req.UID, req.Operation, req.UserInfo)

	// determine whether to perform mutation
	if !mutationRequired(req, &ignoredNamespaces, &pod, whsvr.rewriteConfig) {
		glog.Infof("Skipping mutation for %s/%s (%s) due to policy check", req.Namespace, pod.Name, pod.GenerateName)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	// Workaround: https://github.com/kubernetes/kubernetes/issues/57982
	annotations := make(map[string]string)
	patchBytes, err := createPatch(&pod, whsvr.rewriteConfig, annotations)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	glog.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

// Serve method for webhook server
func (whsvr *WebhookServer) serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		glog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = whsvr.mutate(&ar)
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	glog.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}
