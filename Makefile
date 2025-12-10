.PHONY: help build dev clean docker-build docker-run docker-stop apply-crds delete-crds apply-examples delete-examples logs shell

# Default target
.DEFAULT_GOAL := help

# Variables
IMAGE_NAME := cloudflare-kubernetes-operator
IMAGE_TAG := latest
DOCKER_IMAGE := $(IMAGE_NAME):$(IMAGE_TAG)

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

build: docker-build ## Alias pour docker-build

dev: ## Lance l'opérateur en mode développement (rebuild à chaque démarrage)
	@echo "$(CYAN)🚀 Démarrage en mode développement...$(NC)"
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)⚠ Fichier .env non trouvé, copie depuis .env.example$(NC)"; \
		cp .env.example .env; \
		echo "$(RED)❌ Veuillez éditer le fichier .env avec vos valeurs$(NC)"; \
		exit 1; \
	fi
	docker-compose up --build

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

docker-stop: ## Arrête et nettoie les conteneurs
	@echo "$(CYAN)🛑 Arrêt de l'opérateur...$(NC)"
	docker-compose down
	@echo "$(GREEN)✓ Opérateur arrêté$(NC)"

logs: ## Affiche les logs du conteneur en cours d'exécution
	@echo "$(CYAN)📋 Logs de l'opérateur:$(NC)"
	docker logs -f cloudflare-operator 2>/dev/null || echo "$(RED)❌ Conteneur non démarré$(NC)"

shell: ## Ouvre un shell dans le conteneur
	@echo "$(CYAN)🐚 Ouverture d'un shell dans le conteneur...$(NC)"
	docker exec -it cloudflare-operator /bin/sh || echo "$(RED)❌ Conteneur non démarré. Utilisez 'make docker-run' d'abord$(NC)"

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

apply-examples: ## Applique les exemples de ressources
	@echo "$(CYAN)📝 Application des exemples...$(NC)"
	kubectl apply -f examples/
	@echo "$(GREEN)✓ Exemples appliqués$(NC)"
	@echo "$(CYAN)💡 Vérifiez avec: kubectl get cloudflarerecords,cloudflare-rulesets$(NC)"

delete-examples: ## Supprime les exemples de ressources
	@echo "$(CYAN)🗑️  Suppression des exemples...$(NC)"
	kubectl delete -f examples/ --ignore-not-found
	@echo "$(GREEN)✓ Exemples supprimés$(NC)"

status: ## Affiche le statut des ressources Cloudflare
	@echo "$(CYAN)📊 Statut des ressources Cloudflare:$(NC)"
	@echo ""
	@echo "$(YELLOW)CloudflareRecords:$(NC)"
	kubectl get cloudflarerecords -A -o wide 2>/dev/null || echo "  Aucune ressource trouvée"
	@echo ""
	@echo "$(YELLOW)CloudflareRulesets:$(NC)"
	kubectl get cloudflare-rulesets -A -o wide 2>/dev/null || echo "  Aucune ressource trouvée"

describe-records: ## Décrit toutes les CloudflareRecords
	@echo "$(CYAN)📋 Description des CloudflareRecords:$(NC)"
	kubectl describe cloudflarerecords -A

describe-rulesets: ## Décrit tous les CloudflareRulesets
	@echo "$(CYAN)📋 Description des CloudflareRulesets:$(NC)"
	kubectl describe cloudflare-rulesets -A

watch: ## Surveille les changements des ressources en temps réel
	@echo "$(CYAN)👀 Surveillance des ressources (Ctrl+C pour arrêter)...$(NC)"
	kubectl get cloudflarerecords,cloudflare-rulesets -A --watch

test: ## Lance les tests dans Docker (à implémenter)
	@echo "$(YELLOW)⚠ Tests non encore implémentés$(NC)"

lint: ## Vérifie le code avec ESLint dans Docker (à configurer)
	@echo "$(YELLOW)⚠ Linting non encore configuré$(NC)"

rebuild: clean docker-build ## Nettoie et reconstruit l'image Docker
	@echo "$(GREEN)✓ Rebuild complet terminé$(NC)"
