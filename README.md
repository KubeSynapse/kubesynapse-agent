# KubeSynapse Agent 🚀

KubeSynapse is an intelligent Kubernetes AIOps agent designed to automate incident response and cluster maintenance. It monitors cluster health, detects anomalies, and can perform automated remedial actions based on predefined policies.

## Features ✨

- **Intelligent Monitoring**: Real-time analysis of pod restarts, crash loops, and node health.
- **Automated Remediation**: Configurable thresholds for automated pod restarts and resource cleanup.
- **Helm-Native Deployment**: Easy to install and manage using standard Kubernetes tooling.
- **Enterprise Ready**: Designed for scalability and observability with detailed logging and metrics.

## Getting Started 🛠️

### Prerequisites

- A running Kubernetes cluster.
- `kubectl` and `helm` installed and configured.
- Go 1.26+ (for building from source).

### Quick Setup

Use the provided setup script to deploy KubeSynapse to your cluster:

```bash
chmod +x setup.sh
./setup.sh
```

### Manual Installation (Helm)

**From Local Path:**
```bash
helm upgrade --install kubesynapse ./charts/kubesynapse --namespace synapse --create-namespace
```

**From Official Repository:**
```bash
helm repo add kubesynapse https://KubeSynapse.github.io/kubesynapse-agent/
helm repo update
helm upgrade --install kubesynapse kubesynapse/kubesynapse --namespace synapse --create-namespace
```

## Repository Structure 📂

- `kubesynapse-agent/`: Core Go application logic.
- `charts/`: Helm charts for Kubernetes deployment.
- `setup.sh`: Interactive setup script for rapid deployment.

## Development 💻

To build the project locally:

```bash
cd kubesynapse-agent
make build
```

To run tests:

```bash
make test
```

## License 📄

[MIT License](LICENSE)
