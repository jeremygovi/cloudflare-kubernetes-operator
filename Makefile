.PHONY: help clean docker-build docker-run apply-crds delete-crds

# Default target
.DEFAULT_GOAL := help

# Variables
IMAGE_NAME := cloudflare-kubernetes-operator
IMAGE_TAG := latest
DOCKER_IMAGE := $(IMAGE_NAME):$(IMAGE_TAG)
KIND_CLUSTER_NAME := cloudflare-operator

# Colors for output
CYAN := \033[0;36m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

help: ## Affiche l'aide et la liste des commandes disponibles
	@echo "$(CYAN)┌─────────────────────────────────────────────────────────────┐$(NC)"
	@echo "$(CYAN)│  Cloudflare Kubernetes Operator - Makefile Commands        │$(NC)"
	@echo "$(CYAN)└─────────────────────────────────────────────────────────────┘$(NC)"
	@echo ""
	@echo "$(GREEN)Commandes Docker (tout s'exécute dans Docker):$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(GREEN)Variables d'environnement requises:$(NC)"
	@echo "  CLOUDFLARE_API_TOKEN    Token API Cloudflare"
	@echo ""
	@echo "$(GREEN)Exemples:$(NC)"
	@echo "  make docker-build       Construire l'image Docker"
	@echo "  make docker-run         Lancer l'opérateur (foreground)"
	@echo "  make shell              Ouvrir un shell dans le conteneur"
	@echo ""

clean: ## Nettoie les conteneurs et images Docker
	@echo "$(CYAN)🧹 Nettoyage des conteneurs et images...$(NC)"
	docker-compose down -v 2>/dev/null || true
	docker rmi $(DOCKER_IMAGE) 2>/dev/null || true
	@echo "$(GREEN)✓ Nettoyage terminé$(NC)"

docker-build: ## Construit l'image Docker
	@echo "$(CYAN)🐳 Construction de l'image Docker...$(NC)"
	docker build -t $(DOCKER_IMAGE) .
	@echo "$(GREEN)✓ Image Docker construite: $(DOCKER_IMAGE)$(NC)"

docker-run: ## Lance l'opérateur en foreground (Ctrl+C pour arrêter)
	@echo "$(CYAN)🐳 Démarrage de l'opérateur avec Docker Compose...$(NC)"
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)⚠ Fichier .env non trouvé, copie depuis .env.example$(NC)"; \
		cp .env.example .env; \
		echo "$(RED)❌ Veuillez éditer le fichier .env avec vos valeurs$(NC)"; \
		exit 1; \
	fi
	docker-compose up

apply-crds: ## Applique les CRDs dans Kubernetes
	@echo "$(CYAN)📝 Application des CRDs...$(NC)"
	kubectl apply -f crds/cloudflarerecord-crd.yaml
	kubectl apply -f crds/cloudflare-ruleset-crd.yaml
	@echo "$(GREEN)✓ CRDs appliquées$(NC)"

delete-crds: ## Supprime les CRDs de Kubernetes
	@echo "$(CYAN)🗑️  Suppression des CRDs...$(NC)"
	kubectl delete -f crds/cloudflarerecord-crd.yaml --ignore-not-found
	kubectl delete -f crds/cloudflare-ruleset-crd.yaml --ignore-not-found
	@echo "$(GREEN)✓ CRDs supprimées$(NC)"
