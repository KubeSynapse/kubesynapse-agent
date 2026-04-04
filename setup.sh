#!/bin/bash

# =============================================================================
# KubeSynapse — End-to-End Setup Script (Helm Version)
# =============================================================================

set -e

INFO="ℹ️ "
SUCCESS="✅ "
ROCKET="🚀 "
WAIT="⌛ "

echo -e "$ROCKET Starting Final KubeSynapse Enterprise Deployment..."

# 1. Namespace Creation
echo -e "$INFO Creating 'synapse' namespace..."
kubectl create namespace synapse --dry-run=client -o yaml | kubectl apply -f -

# 2. Build Agent
echo -e "$INFO Building KubeSynapse Agent..."
(cd kubesynapse-agent && make docker-build)

# 3. Deploy KubeSynapse Agent (Stateless)
echo -e "$INFO Deploying KubeSynapse Agent via Helm..."
helm upgrade --install kubesynapse ./charts/kubesynapse --namespace synapse --create-namespace

# 4. Summary
echo -e "\n$SUCCESS KubeSynapse Agent is ONLINE!"
