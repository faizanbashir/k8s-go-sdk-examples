package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sClient struct {
	Client kubernetes.Interface
}

// stringPatch specifies a patch operation for a string.
type stringPatch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func getK8sClient() *K8sClient {
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
	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		err := fmt.Errorf("error getting kubernetes config: %v\n", err)
		log.Fatal(err.Error)
	}

	fmt.Printf("%T\n", client)
	return &K8sClient{
		Client: client,
	}
}

func main() {
	client := getK8sClient()

	deploymentName := ""
	namespace := ""
	// Creating a new Deployment
	image := "docker.io/httpd:latest"
	err := client.CreateDeployment(deploymentName, namespace, image, 1)
	if err != nil {
		log.Fatal(err)
	}

	// Get a deployment
	deployment, err := client.GetDeployment(deploymentName, namespace)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v\n", deployment)

	// Listing deployment
	deployment, err := client.ListDeployment(namespace)
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range deployment.Items {
		fmt.Printf("%s\n", d.ObjectMeta.Name)
	}

	// Updating the image for Deployment
	client.UpdateDeployment(deploymentName, namespace, "docker.io/nginx:latest")

	// Scaling Deployment
	client.ScaleDeployment(deploymentName, namespace, 1)

	// Deleting Deployment
	err := client.DeleteDeployment(deploymentName, namespace)
	if err != nil {
		log.Fatal(err)
	}
}

func (c *K8sClient) CreateDeployment(name, namespace, image string, replicas int32) error {
	deploymentObject := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "web",
							Image: image,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	// Create Deployment
	fmt.Println("Creating deployment...")
	result, err := c.Client.AppsV1().Deployments(namespace).Create(context.TODO(), deploymentObject, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("Created deployment %q.\n", result.GetName())
	return nil
}

func (c *K8sClient) GetDeployment(name, namespace string) (*appsv1.Deployment, error) {
	fmt.Println("Get Deployment in namespace", namespace)
	result, err := c.Client.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})

	if err != nil {
		fmt.Printf("Failed to getting Deployment: %v\n", err)
		return nil, err
	}
	return result, nil
}

func (c *K8sClient) ListDeployment(namespace string) (*appsv1.DeploymentList, error) {
	fmt.Println("List Deployments")
	deployments, err := c.Client.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("error listing deployments: %v\n", err)
		return nil, err
	}
	return deployments, nil
}

func (c *K8sClient) UpdateDeployment(name, namespace, image string) {
	fmt.Printf("Updating Deployment `%s` in namespace `%s`\n", name, namespace)
	payload := []stringPatch{{
		Op:    "replace",
		Path:  "/spec/template/spec/containers/0/image",
		Value: image,
	}}
	payloadBytes, _ := json.Marshal(payload)

	_, err := c.Client.AppsV1().Deployments(namespace).Patch(context.TODO(), name, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
	if err != nil {
		err := fmt.Errorf("[x] Error Update Deployment Image: %v\n", err)
		panic(err)
	}

	fmt.Printf("Successfully update image for Deployment to %s\n", name)
}

func (c *K8sClient) ScaleDeployment(name, namespace string, replica int32) {
	scaleObj, err := c.Client.AppsV1().Deployments(namespace).GetScale(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("error getting scale object: %v\n", err)
		os.Exit(1)
	}
	sd := *scaleObj
	if sd.Spec.Replicas == replica || replica < 0 {
		fmt.Printf("Deployment %s replicas %d, no changes applied\n", name, replica)
		return
	} else if sd.Spec.Replicas > replica {
		fmt.Printf("Scale down Deployment %s from %d to %d replicas\n", name, sd.Spec.Replicas, replica)
	} else {
		fmt.Printf("Scale Up Deployment %s from %d to %d replicas\n", name, sd.Spec.Replicas, replica)
	}
	sd.Spec.Replicas = replica
	scaleDeployment, err := c.Client.AppsV1().Deployments(namespace).UpdateScale(context.Background(), name, &sd, metav1.UpdateOptions{})
	if err != nil {
		fmt.Printf("error updating scale object: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Successfully scaled deployment %s to %d replicas", name, scaleDeployment.Spec.Replicas)
}

func (c *K8sClient) DeleteDeployment(name, namespace string) error {
	fmt.Printf("Deleting Deployment `%s` in namespace `%s`\n", name, namespace)
	err := c.Client.AppsV1().Deployments(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		err := fmt.Errorf("[x] error deleting Deployment: %v\n", err)
		return err
	}
	fmt.Printf("Successfully deleted Deployment %s\n", name)
	return nil
}

func int32Ptr(i int32) *int32 {
	return &i
}
