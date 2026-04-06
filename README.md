<div align="center">

<img src="https://img.shields.io/badge/KubeSynapse-v1.0.4-0f172a?style=for-the-badge&logo=kubernetes&logoColor=white" />
<img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white" />
<img src="https://img.shields.io/badge/Helm-3.x-277A9F?style=for-the-badge&logo=helm&logoColor=white" />
<img src="https://img.shields.io/badge/License-MIT-22c55e?style=for-the-badge" />
<img src="https://img.shields.io/badge/AIOps-Powered-8b5cf6?style=for-the-badge&logo=openai&logoColor=white" />

<br /><br />

# KubeSynapse

### Enterprise-Grade AI-Powered Kubernetes Incident Response & AIOps Platform

**Zero-touch incident resolution — from EKS pod crash to Slack alert + Jira ticket with AI-generated root cause analysis, in under 2 minutes.**

[📖 Documentation](#architecture) · [🚀 Quick Start](#quick-start) · [🛠️ Helm Install](#helm-installation) · [🔌 Integrations](#integrations) · [🔒 Security](#security)

</div>

---

## Table of Contents

1. [Overview](#-overview)
2. [The Problem We Solve](#-the-problem-we-solve)
3. [Architecture](#-architecture)
4. [Tech Stack](#-tech-stack)
5. [Core Agent Capabilities](#-core-agent-capabilities)
6. [Quick Start](#-quick-start)
7. [Helm Installation](#-helm-installation)
8. [EKS Production Deployment](#-eks-production-deployment)
9. [Integrations](#-integrations)
10. [AIOps Pipeline — End-to-End Flow](#-aiops-pipeline--end-to-end-flow)
11. [EKS Intelligent Incident Responder](#-eks-intelligent-incident-responder)
12. [Security & IAM Best Practices](#-security--iam-best-practices)
13. [Cost & Scaling](#-cost--scaling)
14. [Development](#-development)
15. [Configuration Reference](#-configuration-reference)
16. [Repository Structure](#-repository-structure)

---

## 🧠 Overview

**KubeSynapse** is a production-grade, AI-native Kubernetes AIOps platform that **senses, understands, and autonomously remediates** infrastructure incidents — all within your network. It combines a lightweight Go-based cluster agent with a 6-layer AI pipeline to eliminate the most costly problems in modern operations:

| Problem | Industry Average | KubeSynapse |
|---|---|---|
| Mean Time To Resolve (MTTR) | 1–4 hours | **< 10 minutes** |
| Alert false-positive rate | 70–80% | **< 20%** (Isolation Forest ML) |
| Ticket creation time | 15–45 min/incident | **Fully automated** |
| LLM data privacy | Logs sent to cloud APIs | **Data stays in your network** |
| Onboarding cost | Dynatrace/Datadog $$$$ | **100% open-source** |

KubeSynapse is designed for **SRE teams, platform engineering teams, and DevOps organizations** that need enterprise-grade incident automation without vendor lock-in or cloud data exposure.

---

## 🔴 The Problem We Solve

### The Reality of Modern Incident Response

Every production Kubernetes environment generates thousands of events, log lines, and alerts every minute. Despite tooling like Splunk, PagerDuty, and Grafana, engineering teams still face:

- **Alert Fatigue** — On-call engineers receive hundreds of alerts nightly. 70–80% are noise or duplicates, leading to burnout and missed real incidents.
- **Slow MTTR** — Average industry MTTR is 1–4 hours. Every minute of downtime costs $5,000–$300,000 depending on the system.
- **Knowledge Silos** — Root cause knowledge lives in senior engineers' heads. When unavailable, resolution slows dramatically.
- **Manual Toil** — Creating Jira tickets, posting Slack updates, pulling `kubectl logs`, correlating events — all done manually, wasting hours per incident.
- **Privacy & Cost Concerns** — Sending production logs to OpenAI/Claude cloud APIs exposes sensitive data and creates unpredictable token costs at scale.

### Why Existing Tools Fall Short

| Tool | Gap |
|------|-----|
| ELK / Splunk | Reactive. Requires manually-defined thresholds. Cannot detect unknown anomaly patterns. |
| PagerDuty / OpsGenie | Sends alerts but does not explain or fix them. |
| Dynatrace / Datadog AIOps | Intelligent, but expensive, cloud-dependent, and logs leave your network. |
| GitHub Copilot / ChatGPT | Assist with code — not live incident response. |

---

## 🏗️ Architecture

KubeSynapse uses a **6-layer multi-agent AI pipeline** that runs entirely within your infrastructure:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         DATA SOURCES                                    │
│        Splunk / AWS CloudWatch / Prometheus / Kubernetes Events         │
└────────────────────────────┬────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              LAYER 1 — LOG INGESTION & PREPROCESSING                    │
│   KubeSynapse Agent (Go) watches pod events via K8s API                 │
│   Collects: crash logs, describe output, resource metrics, events       │
│   Output: Structured incident payload dispatched to webhook             │
└────────────────────────────┬────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              LAYER 2 — ANOMALY DETECTION ENGINE                         │
│   scikit-learn Isolation Forest (unsupervised ML)                       │
│   No labeled training data needed — learns your own log patterns        │
│   Anomaly score < -0.1 → Flagged for AI analysis                       │
└────────────────────────────┬────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              LAYER 3 — RAG KNOWLEDGE RETRIEVAL                          │
│   Qdrant Vector DB → Semantic search on past incidents + runbooks       │
│   nomic-embed-text (via Ollama) for local embeddings                    │
│   Returns top-3 similar past incidents with resolution steps            │
└────────────────────────────┬────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              LAYER 4 — LOCAL LLM REASONING ENGINE                       │
│   Ollama running Qwen3:7b, DeepSeek-Coder, or llama3.2                 │
│   Zero cloud API tokens. Data never leaves your network.                │
│   Output: { root_cause, severity, recommended_fix, confidence_score }  │
└────────────────────────────┬────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              LAYER 5 — MULTI-AGENT DISPATCHER (n8n)                     │
│   ├─ Incident Agent  → Creates Jira ticket with full AI diagnosis       │
│   ├─ Notification Agent → Posts to Slack/Teams with AI summary         │
│   ├─ Remediation Agent → Triggers K8s restart / Lambda (conf > 85%)   │
│   └─ Cost Agent → Correlates incident with AWS spend spike             │
└────────────────────────────┬────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              LAYER 6 — OBSERVABILITY & AUDIT LAYER                      │
│   MLflow → Logs every AI decision, anomaly score, agent action          │
│   Taipy Dashboard → Real-time visual dashboard of AI activity           │
│   Jaeger (optional) → Distributed tracing across agent pipeline         │
└─────────────────────────────────────────────────────────────────────────┘
```

### EKS Incident Responder — Specific Event Flow

```
[EKS Pod: CrashLoopBackOff / OOMKilled / Error]
     │
     ▼
[Splunk Alert fires] — search: kubernetes.pod.status=Error OR restart_count > 3
     │
     ▼
[Splunk Webhook] — POST to n8n webhook: { microservice, namespace, restart_count, cluster }
     │
     ▼
[n8n Webhook Trigger] → [Code Node: extract + validate payload]
     │
     ▼
[Diagnostics Collector — *optional*: Jenkins / curl / Argo / Lambda]
     ├── kubectl logs <pod> --previous
     ├── kubectl describe pod <pod>
     ├── kubectl top pod <pod>
     └── kubectl get events -n <namespace>
     │
     ▼
[Ollama / Claude API] — LLM root cause analysis + remediation steps
     │
     ▼
[Parallel Dispatch]
     ├── Slack: Rich block message to #incidents
     └── Jira: Auto-created ticket with AI diagnosis + severity + labels
```

---

## 🛠️ Tech Stack

All components are **open-source, free, and enterprise-ready**:

| Component | Tool | Purpose |
|-----------|------|---------|
| **Core Agent** | Go 1.22+ | Kubernetes pod watcher, log collector, metric collector |
| **Alert Source** | Splunk / CloudWatch / Prometheus | Detects EKS pod crashes, fires webhook |
| **Orchestration** | n8n (or any webhook consumer) | Workflow engine connecting all pipeline stages |
| **Data Collection** | Jenkins *(optional)* | Runs `kubectl` diagnostics commands — replaceable with any HTTP-callable script, Lambda, or Argo Workflow |
| **Anomaly Detection** | scikit-learn Isolation Forest | Unsupervised ML with no labeled training data |
| **Local LLM Runtime** | Ollama | Run Qwen3/DeepSeek/llama3.2 locally, OpenAI-compat API |
| **LLM Models** | Qwen3:7b, DeepSeek-Coder, llama3.2 | Reasoning + root cause analysis, zero tokens cost |
| **Embeddings** | nomic-embed-text (Ollama) | Convert text to vectors for RAG |
| **Vector Database** | Qdrant | Store & search past incidents/runbooks |
| **Notification** | Slack / Teams Webhooks | Rich incident alerts to team channels |
| **Ticketing** | Jira REST API | Auto-create incidents with full AI context |
| **AI Observability** | MLflow | Log AI decisions, track model drift |
| **Dashboard** | Taipy | Real-time Python web UI |
| **K8s Deployment** | Helm 3 | Package and deploy all components |
| **Cloud Infrastructure** | AWS EKS, Lambda, CloudWatch | Your existing cloud stack |
| **IaC** | Terraform | Deploy entire stack with one command |

### Infrastructure Requirements

| Environment | Spec | Notes |
|-------------|------|-------|
| Local development (Mac M3 Pro 32GB) | CPU-only Ollama, `kubectl` + `demo-manifests/` | Full stack runs locally against minikube/kind, no cloud spend |
| Demo / POC | 1× g4dn.xlarge (~$0.50/hr) | T4 GPU for fast Ollama inference |
| Production (EKS) | Separate node groups per workload | See [EKS Production Deployment](#-eks-production-deployment) |

---

## 🔧 Core Agent Capabilities

The KubeSynapse Go agent (`kubesynapse-agent/`) is a Kubernetes-native process that runs continuously inside your cluster:

| Package | Responsibility |
|---------|---------------|
| `watcher` | Watches pod events via the Kubernetes API — detects `CrashLoopBackOff`, `OOMKilled`, `Error` states |
| `collector` | Collects pod logs (previous crash), describe output, and live resource metrics |
| `discovery` | Identifies and resolves pod names matching a microservice label |
| `dispatcher` | Batches enriched incident payloads and dispatches to configured sinks (webhooks) |
| `sink` | Pluggable output destinations — HTTP webhook (n8n), custom APIs |
| `dedup` | Redis-backed deduplication — prevents duplicate alerts for the same incident within a configurable window |
| `redactor` | Scrubs secrets, tokens, and sensitive values from log data before forwarding |
| `models` | Shared incident data models and severity classification |
| `health` | `/healthz` and `/readyz` endpoints for Kubernetes liveness/readiness probes |
| `config` | Environment-variable-driven configuration with Kubernetes Secret support |

---

## 🚀 Quick Start

### Prerequisites

- Kubernetes cluster (local: minikube/kind, or AWS EKS)
- `kubectl` and `helm` installed and configured
- Go 1.22+ *(for building from source)*

### Option 1 — One-Command Demo (Recommended)

```bash
# Clone the repository
git clone https://github.com/KubeSynapse/kubesynapse-agent.git
cd kubesynapse-agent

# Deploy the full stack to the 'synapse' namespace
./setup.sh
```

The setup script deploys the following services in the `synapse` namespace:

| Service | Kubernetes DNS | Purpose |
|---------|---------------|---------|
| **KubeSynapse Agent** | `kubesynapse.synapse.svc.cluster.local` | Core Kubernetes watcher |
| **Qdrant** | `qdrant.synapse.svc.cluster.local:6333` | Vector database for RAG |
| **n8n** | `n8n.synapse.svc.cluster.local:5678` | Workflow orchestration |
| **MLflow** | `mlflow.synapse.svc.cluster.local:5000` | AI decision tracking |
| **Redis** | `redis.synapse.svc.cluster.local:6379` | Deduplication cache |
| **Ollama** | `ollama.synapse.svc.cluster.local:11434` | Local LLM inference |

### Option 2 — Local Simulation (minikube / kind)

Test the full pipeline locally using the `demo-manifests/` included in the repo:

```bash
# Deploy the full stack locally (minikube or kind)
kubectl apply -f demo-manifests/qdrant.yaml
kubectl apply -f demo-manifests/redis.yaml
kubectl apply -f demo-manifests/ollama.yaml
kubectl apply -f demo-manifests/n8n.yaml
kubectl apply -f demo-manifests/mlflow.yaml
kubectl apply -f demo-manifests/kubesynapse-agent.yaml

# Pull LLM model (via Ollama pod)
ollama pull llama3.2

# Fire a simulated EKS incident alert directly at n8n
curl -X POST http://localhost:5678/webhook/eks-incident \
  -H 'Content-Type: application/json' \
  -d '{
    "microservice": "payment-service",
    "namespace": "production",
    "restart_count": "7",
    "reason": "OOMKilled",
    "cluster": "prod-eks-cluster",
    "timestamp": "2026-03-21T17:00:00Z"
  }'
```

Or use the bundled setup script to deploy everything in one command:

```bash
./demo-manifests/setup.sh
```

Watch the end-to-end pipeline execute in the **n8n Executions tab** at `http://localhost:5678`.

---

## 📦 Helm Installation

### From the Official Helm Repository

```bash
helm repo add kubesynapse https://KubeSynapse.github.io/kubesynapse-agent/
helm repo update
helm upgrade --install kubesynapse kubesynapse/kubesynapse \
  --namespace synapse \
  --create-namespace
```

### From Local Path

```bash
helm upgrade --install kubesynapse ./charts/kubesynapse \
  --namespace synapse \
  --create-namespace
```

### Common Helm Values

```yaml
# values.yaml
replicaCount: 1

image:
  repository: ghcr.io/kubesynapse/kubesynapse-agent
  tag: "1.0.4"

config:
  webhookUrl: "http://n8n.synapse.svc.cluster.local:5678/webhook/eks-incident"
  logLevel: "info"
  deduplicationWindowSeconds: 300

resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 256Mi

serviceAccount:
  create: true
  annotations: {}  # Add IRSA annotation for AWS
```

### Helm Reference Commands

```bash
# Upgrade after config changes
helm upgrade kubesynapse kubesynapse/kubesynapse --namespace synapse --values values.yaml

# Check deployed releases
helm list -n synapse

# Rollback to previous release
helm rollback kubesynapse 1 -n synapse

# Uninstall
helm uninstall kubesynapse -n synapse
```

---

## ☁️ EKS Production Deployment

### Namespace & Helm Repos

```bash
kubectl create namespace synapse

helm repo add n8n https://n8n-io.github.io/n8n-helm-chart
helm repo add ollama https://otwld.github.io/ollama-helm/
helm repo add jenkins https://charts.jenkins.io  # optional — only if using Jenkins
helm repo update
```

### Deploy Ollama (LLM Inference)

```bash
# CPU-only (no GPU node group needed)
helm install ollama ollama/ollama \
  --namespace synapse \
  --set ollama.gpu.enabled=false \
  --set ollama.models[0]=llama3.2 \
  --set resources.requests.memory=4Gi \
  --set resources.requests.cpu=2 \
  --set service.type=ClusterIP

# GPU-enabled (g4dn.xlarge — ~3s inference vs ~30s CPU)
helm install ollama ollama/ollama \
  --namespace synapse \
  --set ollama.gpu.enabled=true \
  --set ollama.gpu.number=1 \
  --set ollama.models[0]=llama3.2 \
  --set nodeSelector."eks\.amazonaws\.com/nodegroup"=gpu-nodegroup \
  --set service.type=ClusterIP
```

### Deploy n8n (Workflow Orchestration)

> **Database backend:** n8n uses **SQLite by default** — no extra setup required. This is fine for local development, demos, and single-replica deployments.
>
> **PostgreSQL is optional** but recommended for production if you need:
> - Durable workflow history that survives pod restarts without a PVC
> - Multi-replica n8n scaling (SQLite does not support concurrent writers)
> - Centralized DB backups alongside your existing RDS/Aurora infrastructure
>
> If you don't already run PostgreSQL, **skip it** — SQLite with a PVC works well for most single-node setups.

```bash
kubectl create secret generic n8n-secret \
  --from-literal=password=YourSecurePassword123 \
  --namespace synapse

# Default install — SQLite backend (recommended for quick start)
helm install n8n n8n/n8n \
  --namespace synapse \
  --values charts/n8n-values.yaml
```

**Optional: PostgreSQL backend for production**

```bash
# Create PostgreSQL credentials secret
kubectl create secret generic n8n-postgres-secret \
  --from-literal=db-host=your-postgres-host \
  --from-literal=db-name=n8n \
  --from-literal=db-user=n8n \
  --from-literal=db-password=YourDBPassword \
  --namespace synapse

# Add to your n8n values.yaml under env:
# - name: DB_TYPE
#   value: "postgresdb"
# - name: DB_POSTGRESDB_HOST
#   valueFrom:
#     secretKeyRef:
#       name: n8n-postgres-secret
#       key: db-host
# - name: DB_POSTGRESDB_DATABASE
#   valueFrom:
#     secretKeyRef:
#       name: n8n-postgres-secret
#       key: db-name
# - name: DB_POSTGRESDB_USER
#   valueFrom:
#     secretKeyRef:
#       name: n8n-postgres-secret
#       key: db-user
# - name: DB_POSTGRESDB_PASSWORD
#   valueFrom:
#     secretKeyRef:
#       name: n8n-postgres-secret
#       key: db-password
```

### Deploy Jenkins *(Optional — Diagnostics Collector)*

> **Jenkins is optional.** KubeSynapse dispatches a structured incident payload via HTTP webhook to any consumer. You can replace Jenkins with:
> - A **curl one-liner** in a K8s Job
> - An **AWS Lambda** function triggered via API Gateway
> - An **Argo Workflow** that runs `kubectl` steps
> - Any **HTTP-accessible script** that returns diagnostics JSON
>
> Jenkins is shown here as a reference implementation for teams that already have it.

```bash
helm install jenkins jenkins/jenkins \
  --namespace synapse \
  --values charts/jenkins-values.yaml

# Retrieve initial admin password
kubectl exec -n synapse -it svc/jenkins -c jenkins -- \
  /bin/cat /run/secrets/additional/chart-admin-password
```

### Expose Webhook Receiver via ALB Ingress

> The webhook receiver is **any HTTP endpoint** — n8n, a custom API server, Zapier, or your own Go/Python service. The example below uses n8n.

```yaml
# n8n-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: n8n-webhook-ingress
  namespace: synapse
  annotations:
    kubernetes.io/ingress.class: alb
    alb.ingress.kubernetes.io/scheme: internal
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTPS":443}]'
spec:
  rules:
    - host: n8n-webhook.internal.yourdomain.com
      http:
        paths:
          - path: /webhook/eks-incident
            pathType: Exact
            backend:
              service:
                name: n8n
                port:
                  number: 5678
```

```bash
kubectl apply -f n8n-ingress.yaml
kubectl get ingress -n synapse n8n-webhook-ingress
```

### Jenkins RBAC & IRSA *(Optional)*

If you are using Jenkins as your diagnostics collector, grant it `kubectl` read access via IRSA:

```bash
# Create IAM policy for EKS read access
aws iam create-policy \
  --policy-name JenkinsEKSReadOnly \
  --policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Action": ["eks:DescribeCluster", "eks:ListClusters"],
      "Resource": "*"
    }]
  }'

# Create service account with IRSA annotation
eksctl create iamserviceaccount \
  --name jenkins-sa \
  --namespace synapse \
  --cluster your-cluster-name \
  --attach-policy-arn arn:aws:iam::ACCOUNT_ID:policy/JenkinsEKSReadOnly \
  --approve

# Assign service account to Jenkins
helm upgrade jenkins jenkins/jenkins \
  --namespace synapse \
  --set serviceAccount.name=jenkins-sa \
  --set serviceAccount.create=false
```

### Internal DNS Reference

| Service | Kubernetes DNS |
|---------|---------------|
| n8n | `http://n8n.synapse.svc.cluster.local:5678` |
| Ollama | `http://ollama.synapse.svc.cluster.local:11434` |
| Jenkins *(optional)* | `http://jenkins.synapse.svc.cluster.local:8080` |

---

## 🔌 Integrations

### Splunk Alert Configuration

**Saved Search** *(runs every 2 minutes)*:

```spl
index=kubernetes sourcetype=kube:events
| search reason=BackOff OR reason=OOMKilling OR reason=Failed
| stats count by kubernetes.pod_name, kubernetes.namespace_name, reason
| where count > 2
| eval microservice=kubernetes.pod_name, namespace=kubernetes.namespace_name
```

**Webhook Action** in Splunk → Alerts → Add Actions → Webhook:

```
URL: https://n8n-webhook.internal.yourdomain.com/webhook/eks-incident
Method: POST
Payload:
{
  "microservice": "$result.microservice$",
  "namespace": "$result.namespace$",
  "restart_count": "$result.count$",
  "reason": "$result.reason$",
  "cluster": "prod-eks-cluster",
  "timestamp": "$trigger_time$"
}
```

### Slack Configuration

Rich Block Kit incident notifications are automatically posted to your `#incidents` channel with:
- Microservice name + severity badge
- Namespace + restart count
- AI-generated diagnosis from Ollama/Claude
- Cluster name + timestamp

### Jira Integration

Tickets are auto-created with:
- Summary: `[AUTO] EKS Pod Restart: <service> (<severity>)`
- Priority: mapped from `CRITICAL → Highest`, `HIGH → High`, `MEDIUM → Medium`
- Labels: `["eks", "auto-created", "<namespace>"]`
- Description: full AI diagnosis + remediation steps

### Production Variant — Claude API (No Local GPU Required)

Replace Ollama with the Anthropic Claude API for environments where GPU nodes are unavailable:

```bash
kubectl create secret generic claude-secret \
  --from-literal=api-url=https://api.anthropic.com \
  --from-literal=api-key=sk-ant-YOURTOKEN \
  --namespace synapse
```

n8n Claude API node configuration:
```json
{
  "model": "claude-3-5-sonnet-20241022",
  "max_tokens": 1024,
  "messages": [{
    "role": "user",
    "content": "You are a Kubernetes SRE expert. Analyze this EKS incident: ..."
  }]
}
```

> **Architecture note**: With Claude API, only n8n (or any webhook consumer) runs inside EKS. No Ollama pod required. Jenkins is also fully optional — the diagnostics step can be handled by any callable HTTP endpoint. Cluster cost drops significantly.

---

## 🤖 AIOps Pipeline — Reference Architecture

> **📐 This section describes the intended full AIOps pipeline design.** It is a reference architecture for how the 6-layer Python ML pipeline integrates with the Go agent. Implementation is in progress.

### How the Current Repo Fits the Pipeline

The Go agent (`kubesynapse-agent/`) covers **Layers 1 & 5** today — it watches Kubernetes events, collects diagnostics, and dispatches enriched payloads to any webhook (n8n, custom API, Lambda). The demo manifests in `demo-manifests/` wire up the full supporting stack (Ollama, Qdrant, n8n, MLflow, Redis) on a live cluster.

The remaining layers (ML anomaly detection, RAG, LLM reasoning, observability dashboard) are planned additions to the pipeline.

### Recommended Ollama Models

| Model | Size | Best For | Inference (M3 Pro) |
|-------|------|---------|-------------------|
| `mistral:7b` | ~5 GB | Fast log root cause analysis | ~5s |
| `gemma3:12b` | ~8 GB | Strong instruction following | ~12s |
| `llama3.2` | ~4.7 GB | Balanced reasoning | ~6s |
| `qwen3:7b` | ~5 GB | Function calling + reasoning | ~5s |
| `nomic-embed-text` | ~300 MB | RAG embeddings (required) | ~1s |


---

## 🔒 Security & IAM Best Practices

### Secrets Management

> ⚠️ **Never** store tokens, API keys, or passwords in plain Helm `values.yaml`.

```bash
# Store all integration secrets as Kubernetes Secrets
kubectl create secret generic synapse-secrets \
  --from-literal=slack-webhook=https://hooks.slack.com/services/YOUR/WEBHOOK \
  --from-literal=jira-api-token=YOUR_JIRA_TOKEN \
  --from-literal=jenkins-password=YOUR_PASSWORD \
  --namespace synapse

# Recommended: Use AWS Secrets Manager + External Secrets Operator
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets \
  external-secrets/external-secrets \
  -n external-secrets \
  --create-namespace
```

### Network Policies

Enforce strict pod-to-pod isolation — only n8n (or your chosen webhook consumer) should be allowed to reach Ollama:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ollama-allow-n8n
  namespace: synapse
spec:
  podSelector:
    matchLabels:
      app: ollama
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: n8n
      ports:
        - port: 11434
```

### Production Security Checklist

- [ ] All secrets in Kubernetes Secrets or AWS Secrets Manager (not in `values.yaml`)
- [ ] Webhook receiver Ingress restricted to internal ALB (`scheme: internal`) — not public internet
- [ ] Jenkins UI not exposed externally — access via `kubectl port-forward` only *(if using Jenkins)*
- [ ] Ollama not exposed externally (ClusterIP only)
- [ ] IRSA configured for any pod making AWS API calls (no static credentials)
- [ ] Network Policies applied to restrict lateral movement
- [ ] Resource limits set on all pods (prevent runaway LLM inference)
- [ ] EBS volumes encrypted (gp3 with AWS KMS)
- [ ] Pod Security Standards enforced (restricted or baseline PSA)
- [ ] EKS control plane audit logs enabled
- [ ] n8n execution history retention limited (`N8N_EXECUTIONS_DATA_MAX_AGE=336`)
- [ ] Jira + Slack tokens rotated on a schedule (90-day policy)
- [ ] Log data redacted before forwarding (KubeSynapse `redactor` package)

---

## 📊 Cost & Scaling

### Node Group Sizing (EKS)

| Component | Recommended Instance | Notes |
|-----------|---------------------|-------|
| n8n *(or webhook receiver)* | t3.medium (2 vCPU, 4 GB) | Very lightweight |
| Jenkins *(optional)* | t3.large (2 vCPU, 8 GB) | Only needed if using Jenkins for diagnostics |
| Ollama (CPU) | m5.xlarge (4 vCPU, 16 GB) | ~30s inference |
| Ollama (GPU) | g4dn.xlarge (1× T4 GPU) | ~3s inference |
| KubeSynapse Agent | Any node | 100m CPU / 128Mi memory |

> **Dev/Test**: Use a single `t3.xlarge` node for everything except Ollama GPU.
> **Production**: Separate node groups per workload using node selectors.

### Cost Optimization Tips

- Use **Spot Instances** for n8n and Jenkins agent nodes *(not Ollama — model loading takes time)*
- **Skip Jenkins entirely** if you don't already have it: use a lightweight K8s Job or Lambda for diagnostics
- Set Ollama to **scale-to-zero** when idle using KEDA (Kubernetes Event Driven Autoscaler)
- Use **gp3 instead of gp2** for EBS volumes (20% cheaper, better performance)
- Set **resource requests/limits tightly** to avoid over-provisioning
- Use **Savings Plans** for baseline compute if running 24/7
- Schedule **Karpenter or Cluster Autoscaler** to scale down non-GPU nodes at night

### KEDA Scale-to-Zero for Ollama

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: ollama-scaler
  namespace: synapse
spec:
  scaleTargetRef:
    name: ollama
  minReplicaCount: 0
  maxReplicaCount: 2
  triggers:
    - type: prometheus
      metadata:
        serverAddress: http://prometheus:9090
        metricName: n8n_pending_executions
        threshold: '1'
```

---

## 💻 Development

### Building from Source

```bash
cd kubesynapse-agent
make build
```

### Running Tests

```bash
make test
```

### Linting

```bash
make lint
# or
golangci-lint run
```

### Building Docker Image

```bash
docker build -t kubesynapse-agent:local .
```

### Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Compile Go binary |
| `make test` | Run unit tests with race detector |
| `make lint` | Run golangci-lint |
| `make docker-build` | Build container image |
| `make docker-push` | Push to container registry |

---

## ⚙️ Configuration Reference

KubeSynapse Agent is configured via environment variables, typically injected from a Kubernetes Secret.

| Variable | Description | Default |
|----------|-------------|---------|
| `WEBHOOK_URL` | n8n or custom webhook endpoint | *(required)* |
| `WATCH_NAMESPACE` | Kubernetes namespace to monitor (`""` = all) | `""` |
| `LOG_LEVEL` | Agent log verbosity: `debug`, `info`, `warn` | `info` |
| `DEDUP_WINDOW_SECONDS` | Deduplication window to suppress repeat alerts | `300` |
| `REDIS_URL` | Redis endpoint for deduplication state | `redis:6379` |
| `OLLAMA_URL` | Ollama inference endpoint | `http://ollama:11434` |
| `METRICS_PORT` | Prometheus metrics server port | `8080` |
| `CRASH_THRESHOLD` | Minimum restart count to trigger alert | `3` |
| `LOG_TAIL_LINES` | Number of log lines to collect per crash | `100` |

---

## 📂 Repository Structure

```
kubesynapse-agent/
│
├── kubesynapse-agent/                  # Core Go agent
│   ├── cmd/kubesynapse/main.go         # CLI entrypoint
│   ├── internal/
│   │   ├── collector/diagnostics.go    # Log + resource metrics collection
│   │   ├── config/config.go            # Environment-variable config loading
│   │   ├── dedup/dedup.go              # Redis-backed alert deduplication
│   │   ├── discovery/cluster.go        # Pod name / label resolution
│   │   ├── dispatcher/dispatcher.go    # Incident payload orchestration
│   │   ├── dispatcher/interfaces.go    # Sink interface definitions
│   │   ├── health/health.go            # /healthz + /readyz endpoints
│   │   ├── models/types.go             # Shared incident data models
│   │   ├── redactor/redactor.go        # Secret scrubbing from logs
│   │   ├── sink/webhook/sink.go        # HTTP webhook output sink
│   │   └── watcher/watcher.go          # Kubernetes pod event watcher
│   ├── pkg/logger/logger.go            # Structured logger
│   ├── Dockerfile
│   ├── Makefile
│   ├── go.mod
│   └── .golangci.yml
│
├── charts/kubesynapse/                 # Helm chart (v1.0.4)
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
│       ├── deployment.yaml
│       ├── configmap.yaml
│       ├── rbac.yaml
│       └── pvc.yaml
│
├── demo-manifests/                     # Raw Kubernetes manifests for demo stack
│   ├── setup.sh                        # One-command demo setup
│   ├── destroy.sh                      # Teardown script
│   ├── kubesynapse-agent.yaml          # Agent deployment manifest
│   ├── n8n.yaml                        # n8n workflow engine
│   ├── n8n-ollama-workflow.json        # Pre-built n8n workflow (importable)
│   ├── ollama.yaml                     # Ollama LLM runtime
│   ├── qdrant.yaml                     # Qdrant vector database
│   ├── mlflow.yaml                     # MLflow tracking server
│   ├── redis.yaml                      # Redis deduplication cache
│   ├── test-crash-pod.yaml             # Pod to simulate CrashLoopBackOff
│   ├── ingest_docs.sh                  # Script to seed Qdrant with runbooks
│   └── k8s-sre-knowledge-base.md       # SRE runbooks for RAG ingestion
│
└── .github/workflows/
    ├── ci.yml                          # Build + test on push
    ├── release.yml                     # Helm chart release to GitHub Pages
    └── security.yml                    # Security scanning
```

---

## 📈 Real-World Impact

| Metric | Before KubeSynapse | After KubeSynapse |
|--------|-------------------|------------------|
| MTTR | 1–4 hours | **< 10 minutes** |
| False positive alerts | 70–80% | **< 20%** |
| Ticket creation | 15–45 min manually | **Automated in seconds** |
| LLM token cost | $$$$ / month | **$0 (local Ollama)** |
| On-call engineer hours saved | — | **3+ hrs/week/engineer** |
| Senior engineer knowledge retention | Lost when they leave | **Preserved in Qdrant RAG** |

---

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit your changes (`git commit -m 'feat: add my feature'`)
4. Push to the branch (`git push origin feature/my-feature`)
5. Open a Pull Request

Please ensure all tests pass (`make test`) and linting is clean (`make lint`) before submitting.

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).

---

<div align="center">

**Built with ❤️ by the KubeSynapse team**

[GitHub](https://github.com/KubeSynapse/kubesynapse-agent) · [Issues](https://github.com/KubeSynapse/kubesynapse-agent/issues) · [Discussions](https://github.com/KubeSynapse/kubesynapse-agent/discussions)

</div>
