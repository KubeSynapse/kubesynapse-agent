.PHONY: build test docker-build deploy clean run help

# Variables
BINARY_NAME := kubesynapse
IMAGE_NAME := kubesynapse-agent
IMAGE_TAG  := 1.0.0
NAMESPACE  := aiops

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the Go binary
	@echo "🔨 Building $(BINARY_NAME)..."
	go build -ldflags="-w -s" -o bin/$(BINARY_NAME) ./cmd/kubesynapse/
	@echo "✅ Binary: bin/$(BINARY_NAME)"

test: ## Run unit tests
	@echo "🧪 Running tests..."
	go test -v -race ./...

test-cover: ## Run tests with coverage
	@echo "🧪 Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report: coverage.html"

lint: ## Run linter (requires golangci-lint)
	golangci-lint run ./...

docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "✅ Image: $(IMAGE_NAME):$(IMAGE_TAG)"

docker-push: ## Push Docker image (set REGISTRY first)
	docker push $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

deploy: ## Deploy to Kubernetes cluster
	@echo "🚀 Deploying to namespace $(NAMESPACE)..."
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/rbac.yaml
	kubectl apply -f k8s/configmap.yaml
	kubectl apply -f k8s/deployment.yaml
	@echo "✅ Deployed. Check: kubectl get pods -n $(NAMESPACE)"

deploy-test-pods: ## Deploy test crash pods for integration testing
	@echo "💥 Deploying crash test pods..."
	kubectl apply -f scripts/test-crash-pod.yaml
	@echo "✅ Test pods deployed. Watch watcher logs with: kubectl logs -n $(NAMESPACE) -l app=$(IMAGE_NAME) -f"

undeploy: ## Remove all resources from cluster
	@echo "🗑️  Removing resources..."
	kubectl delete -f k8s/deployment.yaml --ignore-not-found
	kubectl delete -f k8s/configmap.yaml --ignore-not-found
	kubectl delete -f k8s/rbac.yaml --ignore-not-found
	kubectl delete -f scripts/test-crash-pod.yaml --ignore-not-found
	@echo "✅ Resources removed"

run: ## Run locally (requires kubeconfig and services)
	@echo "🚀 Running locally..."
	CLUSTER_NAME=local-dev \
	WATCH_NAMESPACES=default \
	WEBHOOK_URL=http://localhost:5678/webhook/k8s-incident \
	RESTART_THRESHOLD=2 \
	COOLDOWN_SECONDS=60 \
	LOG_LEVEL=debug \
	go run ./cmd/kubesynapse/

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	rm -rf bin/ coverage.out coverage.html
	@echo "✅ Clean"

tidy: ## Tidy Go modules
	go mod tidy

deps: ## Download dependencies
	go mod download
