#!/bin/bash

# =============================================================================
# KubeSynapse Demo - Cleanup & Destroy Script
# =============================================================================

# Colors for pretty printing
RED='\033[0;31m'
INFO='\033[0;36m'
SUCCESS='\033[0;32m'
RESET='\033[0m'

echo -e "$RED🚨  WARNING: This will delete the entire KubeSynapse Stack and all PERSISTENT DATA!$RESET"
read -p "Are you sure you want to continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]
then
    echo -e "$SUCCESS Aborted.$RESET"
    exit 1
fi

echo -e "$INFO 🧹 Cleaning up KubeSynapse environment...$RESET"

# 1. Delete the namespace (this automatically deletes deployments, services, pods, and PVCs)
echo -e "$INFO ℹ️  Deleting 'synapse' namespace (this may take a minute)...$RESET"
kubectl delete namespace synapse --wait=true 2>/dev/null || echo -e "  Namespace 'synapse' already deleted."

# 2. Delete Cluster-Level resources (they are NOT in a namespace, so they must be deleted manually)
echo -e "$INFO ℹ️  Deleting cluster-wide RBAC roles...$RESET"
kubectl delete clusterrolebinding kubesynapse-agent 2>/dev/null
kubectl delete clusterrole kubesynapse-agent 2>/dev/null

# 3. Final Cleanup (Test pods in default namespace if they exist)
echo -e "$INFO ℹ️  Cleaning up leftover test pods...$RESET"
kubectl delete pod crash-test-oom crash-test-loop crash-test-intermittent --namespace default 2>/dev/null

echo -e "\n$SUCCESS ✅  KubeSynapse Environment Cleaned Successfully!$RESET"
echo -e "$INFO All volumes, AI models, and vector data have been removed.$RESET\n"
