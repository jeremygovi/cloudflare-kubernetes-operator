# Cloudflare Kubernetes Operator (Go)

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.29+-326CE5?style=flat&logo=kubernetes)](https://kubernetes.io)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

Opérateur Kubernetes natif en Go pour gérer les ressources Cloudflare (DNS records, Rulesets, etc.) directement depuis vos clusters Kubernetes.

## 🚀 Migration TypeScript → Go

Ce projet a été migré de TypeScript vers Go en utilisant **Kubebuilder** et **controller-runtime** pour bénéficier :

- ✅ Modèle natif des opérateurs Kubernetes (informers, reconcile loop)
- ✅ Finalizers pour un cleanup propre
- ✅ Status subresource et conditions
- ✅ Gestion automatique des requeue avec backoff
- ✅ Idempotence stricte
- ✅ Performance et fiabilité accrues

### Différences clés TypeScript vs Go

| Fonctionnalité | TypeScript (avant)                    | Go (maintenant)                     |
| -------------- | ------------------------------------- | ----------------------------------- |
| Watch API      | Manuel avec `@kubernetes/client-node` | Informers natifs controller-runtime |
| Finalizers     | ❌ Manquant                           | ✅ Implémentés                      |
| Status updates | Basique                               | Conditions + ObservedGeneration     |
| Retry/backoff  | Manuel                                | Automatique avec RequeueAfter       |
| Tests          | ❌ Absents                            | ✅ Ginkgo + envtest                 |
| Génération CRD | Manuelle YAML                         | Automatique avec markers            |

## 📦 Ressources gérées

### CloudflareRecord (DNS Records)

Gestion complète des enregistrements DNS Cloudflare :

- Types supportés : `A`, `AAAA`, `CNAME`, `TXT`, `MX`, `NS`, `SRV`, `CAA`
- Proxied/non-proxied
- TTL configurable
- Priorité (MX, SRV)
- Commentaires

### CloudflareRuleset (Rulesets)

Gestion des rulesets Cloudflare :

- Phases multiples (firewall, transform, compression, etc.)
- Rules avec expressions Cloudflare
- Action parameters flexibles
- Enable/disable par règle

## 🛠 Installation

### Prérequis

- Go 1.22+
- Kubernetes cluster 1.29+ (Kind, k3s, Minikube, ou production)
- Cloudflare API Token avec permissions DNS et Rulesets
- kubectl configuré

### Installation rapide

```bash
# 1. Cloner le repository
git clone https://github.com/jeremygovi/cloudflare-kubernetes-operator
cd cloudflare-kubernetes-operator

# 2. Installer les CRDs
make install

# 3. Générer les manifests (si modifications)
make manifests generate

# 4. Build le binaire
make build

# 5. Configurer les credentials Cloudflare
export CLOUDFLARE_API_TOKEN="votre-token-ici"
export CLOUDFLARE_ACCOUNT_ID="votre-account-id"  # Optionnel

# 6. Run localement (développement)
make run
```

### Déploiement dans le cluster

```bash
# Build et push l'image Docker
make docker-build docker-push IMG=your-registry/cloudflare-operator:v1.0.0

# Deploy dans le cluster
make deploy IMG=your-registry/cloudflare-operator:v1.0.0
```

### Développement avec Docker Compose

```bash
# Créer un fichier .env
cat > .env << EOF
CLOUDFLARE_API_TOKEN=your-token-here
CLOUDFLARE_ACCOUNT_ID=your-account-id
EOF

# Lancer avec docker-compose
docker-compose up --build
```

## 📝 Usage

### Créer un DNS Record

```yaml
apiVersion: cloudflare.k8s.io/v1
kind: CloudflareRecord
metadata:
  name: my-app-dns
  namespace: production
spec:
  zoneId: "abc123def456"
  name: "app.example.com"
  type: A
  content: "192.0.2.1"
  ttl: 1
  proxied: true
  comment: "Production app DNS"
```

```bash
kubectl apply -f config/samples/cloudflare_v1_record_a.yaml
kubectl get cloudflarerecord -n production
kubectl describe cloudflarerecord my-app-dns -n production
```

### Créer un Ruleset

```yaml
apiVersion: cloudflare.k8s.io/v1
kind: CloudflareRuleset
metadata:
  name: security-rules
  namespace: production
spec:
  zoneId: "abc123def456"
  name: "Production Security Rules"
  phase: http_request_firewall_custom
  rules:
    - action: block
      expression: '(ip.geoip.country eq "XX")'
      description: "Block malicious countries"
      enabled: true
```

```bash
kubectl apply -f config/samples/cloudflare_v1_ruleset_security.yaml
kubectl get cloudflareruleset -n production
```

### Vérifier le status

```bash
# Voir les status détaillés
kubectl get cloudflarerecord -o wide
kubectl get cloudflareruleset -o wide

# Voir les conditions
kubectl get cloudflarerecord my-app-dns -o jsonpath='{.status.conditions}'

# Logs de l'opérateur
kubectl logs -n cloudflare-operator-system deployment/cloudflare-operator-controller-manager -f
```

## 🧪 Tests

### Tests unitaires

```bash
# Run tous les tests
make test

# Avec coverage
go test ./... -coverprofile=cover.out
go tool cover -html=cover.out
```

### Tests d'intégration (envtest)

Les tests utilisent `envtest` pour simuler un control plane Kubernetes :

```bash
# Setup envtest binaries
make envtest

# Run integration tests
make test
```

### Tests E2E (Kind)

```bash
# Créer un cluster Kind
kind create cluster --name cloudflare-test

# Installer les CRDs
make install

# Deploy l'opérateur
make deploy IMG=cloudflare-operator:latest

# Run les tests E2E
kubectl apply -f config/samples/
kubectl wait --for=condition=Ready cloudflarerecord/example-a-record --timeout=60s
```

## 🔧 Développement

### Structure du projet

```
.
├── api/v1/                          # CRD Types
│   ├── cloudflarerecord_types.go
│   ├── cloudflareruleset_types.go
│   └── groupversion_info.go
├── cmd/
│   └── main.go                      # Point d'entrée
├── internal/controller/             # Controllers
│   ├── cloudflarerecord_controller.go
│   ├── cloudflareruleset_controller.go
│   └── *_test.go
├── config/                          # Manifests Kubernetes
│   ├── crd/                         # CRDs générées
│   ├── rbac/                        # RBAC roles
│   ├── manager/                     # Deployment operator
│   └── samples/                     # Exemples
├── Makefile                         # Commandes build/test/deploy
├── Dockerfile                       # Image multi-stage
└── docker-compose.yaml              # Dev local
```

### Ajouter une nouvelle ressource Cloudflare

```bash
# Utiliser Kubebuilder pour scaffolder
kubebuilder create api --group cloudflare --version v1 --kind CloudflareZone

# Éditer les types dans api/v1/cloudfarezone_types.go
# Implémenter le controller dans internal/controller/

# Régénérer les manifests
make manifests generate

# Tester
make test
```

### Debugging

```bash
# Run avec logs verbeux
go run cmd/main.go --zap-log-level=debug

# Ou via Makefile
LOG_LEVEL=debug make run

# Profiling
go run cmd/main.go --metrics-bind-address=:8080
curl http://localhost:8080/metrics
```

## 🔐 Sécurité

### Credentials Cloudflare

**JAMAIS** commit les tokens dans le code. Utiliser :

1. **Variables d'environnement** (développement)

   ```bash
   export CLOUDFLARE_API_TOKEN="xxx"
   ```

2. **Kubernetes Secrets** (production)

   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: cloudflare-credentials
   type: Opaque
   stringData:
     api-token: "your-token-here"
   ```

3. **External Secrets Operator** (recommandé)
   Synchroniser depuis AWS Secrets Manager, HashiCorp Vault, etc.

### RBAC

L'opérateur nécessite les permissions suivantes :

- `get`, `list`, `watch`, `create`, `update`, `patch`, `delete` sur les CRDs `cloudflarerecords`, `cloudflare rulesets`
- `get`, `update`, `patch` sur les status subresources
- `update` sur les finalizers

Voir `config/rbac/` pour les manifests complets.

## 🚀 Production

### Bonnes pratiques

1. **Leader Election** : Activer pour la HA

   ```bash
   --leader-elect=true
   ```

2. **Resource limits**

   ```yaml
   resources:
     requests:
       memory: "64Mi"
       cpu: "100m"
     limits:
       memory: "256Mi"
       cpu: "500m"
   ```

3. **Monitoring**

   - Prometheus metrics exposées sur `:8080/metrics`
   - Health probes sur `:8081/healthz` et `:8081/readyz`

4. **Logging structuré**
   ```bash
   --zap-encoder=json --zap-log-level=info
   ```

### Observabilité

```bash
# Prometheus metrics
curl http://localhost:8080/metrics | grep cloudflare

# Health checks
curl http://localhost:8081/healthz
curl http://localhost:8081/readyz
```

## 📚 Documentation Cloudflare

- [API Cloudflare](https://developers.cloudflare.com/api/)
- [DNS Records API](https://developers.cloudflare.com/api/operations/dns-records-for-a-zone-list-dns-records)
- [Rulesets API](https://developers.cloudflare.com/ruleset-engine/rulesets-api/)
- [Cloudflare Go SDK](https://github.com/cloudflare/cloudflare-go)

## 🤝 Contributing

Les contributions sont les bienvenues ! Ouvrir une issue ou une PR.

### Workflow de contribution

1. Fork le projet
2. Créer une branche (`git checkout -b feature/amazing-feature`)
3. Commit les changements (`git commit -m 'Add amazing feature'`)
4. Push vers la branche (`git push origin feature/amazing-feature`)
5. Ouvrir une Pull Request

### Guidelines

- Suivre les conventions Go standard (`gofmt`, `golint`)
- Ajouter des tests pour toute nouvelle fonctionnalité
- Mettre à jour la documentation
- Respecter le code of conduct

## 📄 License

Apache License 2.0 - voir [LICENSE](LICENSE)

## 🙏 Remerciements

- [Kubebuilder](https://book.kubebuilder.io/)
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Cloudflare Go SDK](https://github.com/cloudflare/cloudflare-go)
- Communauté Kubernetes

## 📞 Support

- 🐛 **Bugs** : Ouvrir une [issue](https://github.com/jeremygovi/cloudflare-kubernetes-operator/issues)
- 💬 **Questions** : Discussions GitHub
- 📧 **Contact** : jeremy.govi@example.com

---

**Note** : Version Go migrée depuis TypeScript. Pour l'historique de la version TS, voir la branche `legacy-typescript`.
