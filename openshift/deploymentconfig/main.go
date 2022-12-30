package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	appsv1 "github.com/openshift/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
)

// stringPatch specifies a patch operation for a string.
type stringPatch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// integerPatch specifies a patch operation for a uint32.
type integerPatch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value uint32 `json:"value"`
}

func main() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("error getting user home dir: %v\n", err)
		os.Exit(1)
	}
	kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")
	fmt.Printf("Using kubeconfig: %s\n", kubeConfigPath)

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		err := fmt.Errorf("Error getting kubernetes config: %v\n", err)
		log.Fatal(err.Error)
	}
	client, err := v1.NewForConfig(kubeConfig)
	fmt.Printf("%T\n", client)

	if err != nil {
		err := fmt.Errorf("error getting kubernetes config: %v\n", err)
		log.Fatal(err.Error)
	}

	deploymentName := "my-deployment"
	namespace := "my-namespace"
	// Creating a new DeploymentConfig
	image := "docker.io/httpd:latest"
	err = CreateDeploymentConfig(deploymentName, namespace, image, 1, client)
	if err != nil {
		log.Fatal(err)
	}

	// Listing deployment Configs
	deploymentConfigs, err := ListDeploymentConfigs(namespace, client)
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range deploymentConfigs.Items {
		fmt.Printf("%s\n", d.ObjectMeta.Name)
	}

	// Updating the image for DeploymentConfig
	UpdateDeploymentConfigImage(deploymentName, namespace, "docker.io/nginx:latest", client)

	// Scaling DeploymentConfig
	ScaleDeploymentConfig(deploymentName, namespace, 1, client)

	// Deleting DeploymentConfig
	err = DeleteDeploymentConfig(deploymentName, namespace, client)
	if err != nil {
		log.Fatal(err)
	}
}

func CreateDeploymentConfig(name, namespace, image string, replicas int32, client *v1.AppsV1Client) error {
	fmt.Printf("Creating new DeploymentConfig `%s` in namespace `%s`\n", name, namespace)
	dc := &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentConfigSpec{
			Replicas: replicas,
			Selector: map[string]string{
				"app": name,
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
									Protocol:      corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
			Triggers: []appsv1.DeploymentTriggerPolicy{
				{
					Type: appsv1.DeploymentTriggerOnConfigChange,
				},
			},
		},
	}
	dcObj, err := client.DeploymentConfigs(namespace).Create(context.Background(), dc, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("[x] Error creating DC: %v\n", err)
		return err
	}
	fmt.Printf("Successfully created deploymentconfig to `%s` in namespace `%s`\n", dcObj.ObjectMeta.Name, namespace)
	return nil
}

func ListDeploymentConfigs(namespace string, client *v1.AppsV1Client) (*appsv1.DeploymentConfigList, error) {
	fmt.Printf("Listing DeploymentConifgs in namespace `%s`\n", namespace)
	deploymentConfigs, err := client.DeploymentConfigs(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		err := fmt.Errorf("[x] error listing RC/DC: %v\n", err)
		return nil, err
	}
	return deploymentConfigs, nil
}

func ScaleDeploymentConfig(name, namespace string, scale int, client *v1.AppsV1Client) {
	fmt.Printf("Scaling DeploymentConfig `%s` in namespace `%s`\n", name, namespace)
	replicas := uint32(scale)

	payload := []integerPatch{{
		Op:    "replace",
		Path:  "/spec/replicas",
		Value: replicas,
	}}
	payloadBytes, _ := json.Marshal(payload)

	_, err := client.DeploymentConfigs(namespace).Patch(context.TODO(), name, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
	if err != nil {
		err := fmt.Errorf("[x] Error Scaling DC Image: %v\n", err)
		panic(err)
	}

	fmt.Printf("Successfully scaled deploymentconfig to %d replicas\n", replicas)
}

func UpdateDeploymentConfigImage(name, namespace, image string, client *v1.AppsV1Client) {
	fmt.Printf("Updating DeploymentConfig `%s` in namespace `%s`\n", name, namespace)
	payload := []stringPatch{{
		Op:    "replace",
		Path:  "/spec/template/spec/containers/0/image",
		Value: image,
	}}
	payloadBytes, _ := json.Marshal(payload)

	_, err := client.DeploymentConfigs(namespace).Patch(context.TODO(), name, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
	if err != nil {
		err := fmt.Errorf("[x] Error Update DC Image: %v\n", err)
		panic(err)
	}

	fmt.Printf("Successfully update image for deploymentconfig to %s\n", name)
}

func DeleteDeploymentConfig(name, namespace string, client *v1.AppsV1Client) error {
	fmt.Printf("Deleting DeploymentConfig `%s` in namespace `%s`\n", name, namespace)
	err := client.DeploymentConfigs(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		err := fmt.Errorf("[x] error deleting DC: %v\n", err)
		return err
	}
	fmt.Printf("Successfully deleted deploymentconfig %s\n", name)
	return nil
}
