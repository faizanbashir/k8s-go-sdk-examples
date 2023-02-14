package main

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "github.com/openshift/client-go/machine/clientset/versioned/typed/machine/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"path/filepath"
)

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

	// List all Machineset running in a Cluster
	machineSetList, err := client.MachineSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("No Machinesets found in namespace\n")
			os.Exit(0)
		}
		panic(err.Error())
	}

	replicas := uint32(2)

	payload := []integerPatch{{
		Op:    "replace",
		Path:  "/spec/replicas",
		Value: replicas,
	}}
	payloadBytes, _ := json.Marshal(payload)

	fmt.Printf("There are %d Machinesets in the namespace\n", len(machineSetList.Items))
	for _, ms := range machineSetList.Items {
		fmt.Printf("%+v\n", ms.Name)

		// Scale Machinesets using the Patch approach
		_, err := client.MachineSets(ms.Namespace).Patch(context.TODO(), ms.Name, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
		if err != nil {
			err := fmt.Errorf("[x] Err Updating Scale for MachineSet: %v\n", err)
			panic(err)
		}
		fmt.Printf("Updated scale for Machine Set: %s\n", ms.Name)
	}
}
