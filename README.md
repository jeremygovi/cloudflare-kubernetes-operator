# Cloudflare Kubernetes Operator

Opérateur Kubernetes en Go pour gérer automatiquement les ressources Cloudflare (enregistrements DNS et rulesets) directement depuis Kubernetes.

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

## 🔧 Installation

### 1. Cloner le projet

```bash
git clone <repository-url>
cd cloudflare-kubernetes-operator
```

### 2. Configurer les variables d'environnement

```bash
cp .env.example .env
# Éditer .env et renseigner les variables
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

## 📝 Exemples de ressources

### CloudflareRecord - Enregistrement A

```yaml
apiVersion: cloudflare.k8s.io/v1
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
apiVersion: cloudflare.k8s.io/v1
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

## 🤝 Contribution

Les contributions sont les bienvenues! N'hésitez pas à ouvrir des issues ou des pull requests.

## 📄 Licence

MIT

## 🔗 Liens utiles

- [Documentation Cloudflare API](https://developers.cloudflare.com/api/)
- [Kubernetes Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- [Cloudflare TypeScript SDK](https://github.com/cloudflare/cloudflare-typescript)
