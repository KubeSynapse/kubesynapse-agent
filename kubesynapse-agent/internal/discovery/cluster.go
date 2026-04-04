// Package discovery provides utilities for identifying the host Kubernetes cluster.
package discovery

import (
	"context"
	"log"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DiscoverClusterName attempts to identify the human-readable cluster name.
// It prioritizes the CLUSTER_NAME environment variable, then cloud-specific
// labels on the kube-system namespace, then node labels.
func DiscoverClusterName(clientset *kubernetes.Clientset) string {
	// 1. Manual check (Highest priority)
	if name := os.Getenv("CLUSTER_NAME"); name != "" && name != "kubernetes-production" {
		return name
	}

	ctx := context.TODO()

	// 2. Try GKE-specific labels on kube-system namespace
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, "kube-system", metav1.GetOptions{})
	if err == nil {
		if val, ok := ns.Labels["cloud.google.com/gke-cluster"]; ok {
			log.Printf("[discovery] 🕵️ Discovered GKE Cluster Name: %s", val)
			return val
		}
	}

	// 3. Try EKS-specific labels on nodes
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err == nil && len(nodes.Items) > 0 {
		node := nodes.Items[0]
		// EKS often labels nodes with the cluster name
		for k, v := range node.Labels {
			if strings.Contains(k, "alpha.eksctl.io/cluster-name") || 
			   strings.Contains(k, "eks.amazonaws.com/cluster-name") {
				log.Printf("[discovery] 🕵️ Discovered EKS Cluster Name: %s", v)
				return v
			}
		}
	}

	// 4. Fallback to default
	log.Printf("[discovery] ⚠️ Could not auto-discover cluster name. Falling back to default.")
	return "k8s-cluster"
}
