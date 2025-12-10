# Cloudflare Kubernetes Operator

Opérateur Kubernetes en TypeScript pour gérer automatiquement les ressources Cloudflare (enregistrements DNS et rulesets) directement depuis Kubernetes.

## 🚀 Fonctionnalités

- **CloudflareRecord**: Gestion complète des enregistrements DNS Cloudflare (A, AAAA, CNAME, TXT, MX, etc.)
- **CloudflareRuleset**: Gestion des rulesets Cloudflare (firewall, redirections, transformations)
- Synchronisation automatique avec Cloudflare
- Support du statut des ressources
- Gestion des erreurs et retry automatique
- Logs structurés avec Winston

## 📋 Prérequis

- Docker et Docker Compose
- Kubernetes cluster (local ou distant)
- Token API Cloudflare
- kubectl configuré

**Note:** Aucune installation locale de Node.js ou npm n'est requise. Tout s'exécute dans Docker.

## 🔧 Installation

### 1. Cloner le projet

```bash
git clone <repository-url>
cd cloudflare-kubernetes-operator
```

### 2. Configurer les variables d'environnement

```bash
cp .env.example .env
# Éditer .env et ajouter votre CLOUDFLARE_API_TOKEN
```

Pour obtenir un token API Cloudflare:

1. Allez sur https://dash.cloudflare.com/profile/api-tokens
2. Créez un nouveau token avec les permissions:
   - Zone.DNS (Edit)
   - Zone.Rulesets (Edit)
   - Zone.Zone (Read)

### 3. Construire l'image Docker

```bash
make docker-build
```

### 4. Appliquer les CRDs

```bash
make apply-crds
```

## 🏃 Utilisation

### Démarrage avec Docker (foreground)

```bash
# Construire l'image Docker
make docker-build

# Lancer l'opérateur en foreground (les logs s'affichent directement)
# Utilisez Ctrl+C pour arrêter
make docker-run
```

### Mode développement

```bash
# Lance et rebuild automatiquement à chaque changement
make dev
```

### Autres commandes utiles

```bash
# Arrêter et nettoyer les conteneurs
make docker-stop

# Ouvrir un shell dans le conteneur (si en cours d'exécution)
make shell

# Voir l'aide complète
make help
```

## 📝 Exemples de ressources

### CloudflareRecord - Enregistrement A

```yaml
apiVersion: cloudflare.example.com/v1
kind: CloudflareRecord
metadata:
  name: www-example
  namespace: default
spec:
  zoneId: "abcd1234"
  name: "www.example.com"
  type: "A"
  content: "1.2.3.4"
  ttl: 3600
  proxied: true
  comment: "Main website A record"
```

### CloudflareRuleset - Règles de sécurité

```yaml
apiVersion: cloudflare.example.com/v1
kind: CloudflareRuleset
metadata:
  name: security-rules
  namespace: default
spec:
  zoneId: "abcd1234"
  name: "Security Rules"
  description: "Block malicious traffic"
  phase: "http_request_firewall_custom"
  rules:
    - action: "block"
      expression: "(cf.threat_score gt 50)"
      description: "Block high threat score"
      enabled: true
```

Plus d'exemples dans le dossier `examples/`.

### Appliquer les exemples

```bash
# Appliquer tous les exemples
make apply-examples

# Vérifier le statut
make status

# Voir les détails
make describe-records
make describe-rulesets

# Supprimer les exemples
make delete-examples
```

## 🛠️ Commandes Make disponibles

Exécutez `make` ou `make help` pour voir toutes les commandes:

```
Commandes Docker (tout s'exécute dans Docker):
  docker-build         Construit l'image Docker
  docker-run           Lance en foreground (Ctrl+C pour arrêter)
  docker-stop          Arrête et nettoie les conteneurs
  dev                  Lance en mode développement (rebuild auto)
  build                Alias pour docker-build
  clean                Nettoie les conteneurs et images
  shell                Ouvre un shell dans le conteneur
  logs                 Affiche les logs du conteneur
  rebuild              Nettoie et reconstruit l'image

Kubernetes:
  apply-crds           Applique les CRDs
  delete-crds          Supprime les CRDs
  apply-examples       Applique les exemples
  delete-examples      Supprime les exemples
  status               Affiche le statut des ressources
  watch                Surveille les changements en temps réel
```

## 📁 Structure du projet

```
.
├── src/
│   ├── index.ts                    # Point d'entrée
│   ├── operator.ts                 # Logique principale de l'opérateur
│   ├── controllers/
│   │   ├── cloudflare-record-controller.ts
│   │   └── cloudflare-ruleset-controller.ts
│   └── utils/
│       └── logger.ts               # Configuration des logs
├── crds/
│   ├── cloudflarerecord-crd.yaml   # CRD pour les enregistrements DNS
│   └── cloudflare-ruleset-crd.yaml # CRD pour les rulesets
├── examples/
│   ├── cloudflare-record-*.yaml    # Exemples d'enregistrements DNS
│   └── cloudflare-ruleset-*.yaml   # Exemples de rulesets
├── Dockerfile                       # Image Docker multi-stage
├── docker-compose.yaml              # Configuration Docker Compose
├── Makefile                         # Commandes de build et déploiement
├── package.json
└── tsconfig.json
```

## 🔍 Surveillance et débogage

### Voir les logs de l'opérateur

```bash
# Si lancé en foreground avec docker-run, les logs s'affichent directement
make docker-run

# Si vous avez besoin de voir les logs d'un conteneur en cours
make logs

# Avec kubectl (si déployé dans le cluster)
kubectl logs -n cloudflare-operator -l app=cloudflare-operator -f
```

### Ouvrir un shell dans le conteneur

```bash
# Utile pour déboguer ou inspecter l'environnement
make shell
```

### Surveiller les ressources

```bash
# Statut des ressources
make status

# Surveillance en temps réel
make watch

# Description détaillée
kubectl describe cloudflarerecord <name>
kubectl describe cloudflare-ruleset <name>
```

### Vérifier le statut d'une ressource

```bash
kubectl get cloudflarerecords -o yaml
kubectl get cloudflare-rulesets -o yaml
```

Le champ `status` indique:

- `state`: Pending, Active, ou Error
- `recordId` / `rulesetId`: ID Cloudflare de la ressource
- `message`: Message d'état
- `lastSync`: Dernière synchronisation

## 🔐 Sécurité

- L'opérateur s'exécute avec un utilisateur non-root dans le conteneur
- Le token API Cloudflare doit avoir uniquement les permissions nécessaires
- Utilisez des secrets Kubernetes pour stocker le token en production

### Déploiement avec secrets Kubernetes

```bash
kubectl create secret generic cloudflare-token \
  --from-literal=token=your_token_here \
  -n cloudflare-operator
```

## 🚢 Déploiement en production

### Dans un cluster Kubernetes

1. Créer le namespace:

```bash
kubectl create namespace cloudflare-operator
```

2. Créer le secret:

```bash
kubectl create secret generic cloudflare-token \
  --from-literal=token=your_token_here \
  -n cloudflare-operator
```

3. Appliquer les CRDs:

```bash
make apply-crds
```

4. Déployer l'opérateur (manifests à créer selon vos besoins)

## 🤝 Contribution

Les contributions sont les bienvenues! N'hésitez pas à ouvrir des issues ou des pull requests.

## 📄 Licence

MIT

## 🔗 Liens utiles

- [Documentation Cloudflare API](https://developers.cloudflare.com/api/)
- [Kubernetes Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- [Cloudflare TypeScript SDK](https://github.com/cloudflare/cloudflare-typescript)
