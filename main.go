package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	admissionv1 "k8s.io/api/admission/v1"
)

var RegistryWhitelist []string

type WebhookServer struct {
	client kubernetes.Interface
}

func (ws *WebhookServer) admitPod(pod *corev1.Pod) (bool, string) {
	if len(RegistryWhitelist) == 0 {
		return true, ""
	}

	for _, container := range pod.Spec.Containers {
		allowed := false
		for _, registry := range RegistryWhitelist {
			if strings.HasPrefix(container.Image, registry) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false, fmt.Sprintf("container image %s is not from an allowed registry", container.Image)
		}
	}

	for _, container := range pod.Spec.InitContainers {
		allowed := false
		for _, registry := range RegistryWhitelist {
			if strings.HasPrefix(container.Image, registry) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false, fmt.Sprintf("init container image %s is not from an allowed registry", container.Image)
		}
	}

	return true, ""
}

func (ws *WebhookServer) Handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not read request body: %v", err), http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	admissionReview := admissionv1.AdmissionReview{}
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		http.Error(w, fmt.Sprintf("Could not parse admission review: %v", err), http.StatusBadRequest)
		return
	}

	if admissionReview.Request == nil {
		http.Error(w, "Invalid admission review request", http.StatusBadRequest)
		return
	}

	var admissionResponse *admissionv1.AdmissionResponse

	if admissionReview.Request.Kind.Kind == "Pod" {
		var pod corev1.Pod
		if err := json.Unmarshal(admissionReview.Request.Object.Raw, &pod); err != nil {
			admissionResponse = &admissionv1.AdmissionResponse{
				Result: &metav1.Status{
					Message: fmt.Sprintf("Could not unmarshal Pod object: %v", err),
				},
			}
		} else {
			allowed, message := ws.admitPod(&pod)
			admissionResponse = &admissionv1.AdmissionResponse{
				Allowed: allowed,
				Result: &metav1.Status{
					Message: message,
				},
			}
		}
	} else {
		admissionResponse = &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	admissionResponse.UID = admissionReview.Request.UID

	responseAdmissionReview := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: admissionResponse,
	}

	resp, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not marshal response: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("Warning: Error getting cluster config: %v. Will continue without client.", err)
		config = nil
	}

	var client kubernetes.Interface
	if config != nil {
		client, err = kubernetes.NewForConfig(config)
		if err != nil {
			log.Printf("Warning: Error creating Kubernetes client: %v. Will continue without client.", err)
		}
	}

	webhookServer := &WebhookServer{
		client: client,
	}

	var cert, key, whitelist string
	var port int
	var debug bool

	app := cli.NewApp()
	app.Name = "kube-registry-bouncer"
	app.Usage = "webhook endpoint for kube dynamic admission controller"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "cert, c",
			Usage:       "Path to the certificate to use",
			EnvVars:     []string{"KUBE_BOUNCER_CERTIFICATE"},
			Destination: &cert,
		},
		&cli.StringFlag{
			Name:        "key, k",
			Usage:       "Path to the key to use",
			EnvVars:     []string{"KUBE_BOUNCER_KEY"},
			Destination: &key,
		},
		&cli.IntFlag{
			Name:        "port, p",
			Value:       1323,
			Usage:       "Port to listen to",
			EnvVars:     []string{"KUBE_BOUNCER_PORT"},
			Destination: &port,
		},
		&cli.StringFlag{
			Name:        "registry-whitelist, rw",
			Usage:       "Comma separated list of accepted registries",
			EnvVars:     []string{"KUBE_BOUNCER_REGISTRY_WHITELIST"},
			Destination: &whitelist,
		},
		&cli.BoolFlag{
			Name:        "debug",
			Value:       true,
			Usage:       "Enable debug mode",
			EnvVars:     []string{"KUBE_BOUNCER_DEBUG"},
			Destination: &debug,
		},
	}

	app.Action = func(c *cli.Context) error {
		portStr := fmt.Sprintf("%d", port)

		if debug {
			log.Println("Debug mode:", debug)
			log.Println("Running on port:", portStr)
		}

		if whitelist != "" {
			RegistryWhitelist = strings.Split(whitelist, ",")
			log.Println("WARN: The following registries are allowed:")
			for _, registry := range RegistryWhitelist {
				log.Println("	-", registry)
			}
		} else {
			log.Println("WARN: All registries are allowed")
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/validate-registry", webhookServer.Handle)
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Health status: ok\n"))
		})

		server := &http.Server{
			Addr:    ":" + portStr,
			Handler: mux,
		}

		if cert != "" && key != "" {
			log.Printf("Starting server with TLS on port %s", portStr)
			log.Fatal(server.ListenAndServeTLS(cert, key))
		} else {
			log.Printf("Warning: Starting server without TLS on port %s", portStr)
			log.Fatal(server.ListenAndServe())
		}

		return nil
	}

	app.Run(os.Args)
}
