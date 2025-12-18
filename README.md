# Cloudflare Kubernetes Operator

Opérateur Kubernetes en Go pour gérer automatiquement les ressources Cloudflare (zones, enregistrements DNS et rulesets) directement depuis Kubernetes.

## 🚀 Fonctionnalités

- **CloudflareZone**: Gestion complète des zones Cloudflare (création, configuration, nameservers)
- **CloudflareRecord**: Gestion complète des enregistrements DNS Cloudflare (A, AAAA, CNAME, TXT, MX, etc.)
- **CloudflareRuleset**: Gestion des rulesets Cloudflare (firewall, redirections, transformations)
- Synchronisation automatique avec Cloudflare
- Support du statut des ressources
- Gestion des erreurs et retry automatique
- Logs structurés

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
   - Zone.Zone (Edit)
   - Account.Account Settings (Read) - pour CloudflareZone

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

Zone - Création de zone

```yaml
apiVersion: cloudflare.io/v1
kind: CloudflareZone
metadata:
  name: example-zone
  namespace: default
spec:
  name: "example.com"
  # accountId est optionnel - utilise CLOUDFLARE_ACCOUNT_ID si non spécifié
  # accountId: "your-account-id"
  type: "full" # full, partial, or secondary
  jumpStart: true
  paused: false
```

### CloudflareRecord - Enregistrement A

```yaml
apiVersion: cloudflare.io/v1
kind: CloudflareRecord
metadata:
  name: www-example
  namespace: default
spec:
  domain: "example.com" # Le domaine de base (zone Cloudflare)
  name: "www" # Le sous-domaine (ou @ pour root)
  type: "A"
  content: "1.2.3.4"
  ttl: 3600
  proxied: true
  comment: "Main website A record"
```

**Note**: Le champ `domain` permet de spécifier le nom de domaine sans avoir besoin de connaître le `zoneId` à l'avance. L'opérateur résoudra automatiquement le `zoneId` correspondant.

### CloudflareRuleset - Règles de sécurité

```yaml
apiVersion: cloudflare
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
