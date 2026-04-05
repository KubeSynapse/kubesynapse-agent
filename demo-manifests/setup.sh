#!/bin/bash

# =============================================================================
# KubeSynapse — End-to-End Demo Setup Script
# =============================================================================

INFO="ℹ️ "
SUCCESS="✅ "
ROCKET="🚀 "
WAIT="⌛ "

echo -e "$ROCKET Starting KubeSynapse Demo Deployment..."

# 1. Namespace Creation
echo -e "$INFO Creating 'synapse' namespace..."
kubectl create namespace synapse --dry-run=client -o yaml | kubectl apply -f -

# 2. Deploy Demo Services
echo -e "$INFO Deploying Demo Services (Redis, Ollama, Qdrant, n8n, MLflow and kubesynapse agent)..."
kubectl apply -f redis.yaml
kubectl apply -f ollama.yaml
kubectl apply -f qdrant.yaml
kubectl apply -f n8n.yaml
kubectl apply -f mlflow.yaml
kubectl apply -f kubesynapse-agent.yaml
kubectl apply -f test-crash-pod.yaml

# 3. Wait for Critical Services (Ollama & Qdrant)
echo -e "$WAIT Waiting for AI infrastructure to be ready (Ollama & Qdrant)..."
kubectl wait --for=condition=ready pod -l app=ollama -n synapse --timeout=300s
kubectl wait --for=condition=ready pod -l app=qdrant -n synapse --timeout=300s

# 4. Pull AI Models into Ollama
echo -e "$INFO Pulling AI Models into Ollama (this may take a few minutes)..."
OLLAMA_POD=$(kubectl get pods -l app=ollama -n synapse -o name | head -n 1)

# Direct execution - simpler and more robust for Mac scripting
kubectl exec -i $OLLAMA_POD -n synapse -- ollama pull nomic-embed-text || echo "  ⚠️  Failed to pull nomic-embed-text, please try manually: kubectl exec -it $OLLAMA_POD -n synapse -- ollama pull nomic-embed-text"
kubectl exec -i $OLLAMA_POD -n synapse -- ollama pull qwen2.5-coder:3b || echo "  ⚠️  Failed to pull qwen2.5-coder, please try manually: kubectl exec -it $OLLAMA_POD -n synapse -- ollama pull qwen2.5-coder:3b"

# 5. Initialize Qdrant Collection
echo -e "$INFO Initializing Qdrant RAG Collection..."
# Small retry loop for network readiness
for i in {1..15}; do
    RESPONSE=$(curl -s -X PUT "http://localhost:30633/collections/k8s-incidents" \
         -H "Content-Type: application/json" \
         --data '{
            "vectors": {
                "size": 768,
                "distance": "Cosine"
            }
        }')
    if [[ "$RESPONSE" == *"\"status\":\"ok\""* || "$RESPONSE" == *"already exists"* ]]; then
        echo -e "  $SUCCESS Qdrant collection ready."
        break
    fi
    echo -e "  $WAIT Waiting for Qdrant API / host mapping (attempt $i/15)..."
    sleep 3
done

echo -e "\n$SUCCESS ✅  KubeSynapse Demo is ONLINE!"
echo -e "$INFO Dashboards available at:"
echo -e "  - n8n Workflow:  http://localhost:30678"
echo -e "  - Qdrant UI:    http://localhost:30633/dashboard"
echo -e "  - MLflow UI:    http://localhost:30500"
echo -e "  - Ollama API:   http://localhost:31434"
echo -e "\n$INFO Remember to ACTIVATE the workflow in n8n UI! 🤖✨"
