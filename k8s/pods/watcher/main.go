package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

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
		fmt.Printf("Error getting kubernetes config: %v\n", err)
		os.Exit(1)
	}

	client, err := kubernetes.NewForConfig(kubeConfig)

	if err != nil {
		fmt.Printf("error getting kubernetes config: %v\n", err)
		os.Exit(1)
	}

	// Empty string for all namespaces
	namespace := ""
	fmt.Println("Watch Kubernetes Pods in CrashLoopBackOff state")
	watcher, err := client.CoreV1().Pods(namespace).Watch(context.Background(),
		metav1.ListOptions{
			FieldSelector: "",
		})
	if err != nil {
		fmt.Printf("error create pod watcher: %v\n", err)
		return
	}

	for event := range watcher.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			continue
		}
		for _, c := range pod.Status.ContainerStatuses {
			if c.State.Waiting != nil && c.State.Waiting.Reason == "CrashLoopBackOff" {
				fmt.Printf("PodName: %s, Namespace: %s, Phase: %s\n", pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, pod.Status.Phase)
			}
		}
	}
}
