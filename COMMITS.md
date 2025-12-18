# Commits Conventionnels

Ce projet utilise les [Commits Conventionnels](https://www.conventionalcommits.org/fr/) pour normaliser les messages de commit et automatiser la gestion des versions.

## Format des commits

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

## Types de commits

- **feat**: Une nouvelle fonctionnalité (génère un release mineur)
- **fix**: Un correctif de bug (génère un patch)
- **docs**: Modifications de documentation uniquement
- **style**: Changements qui ne modifient pas le code (formatage, etc.)
- **refactor**: Refactorisation du code
- **perf**: Amélioration des performances
- **test**: Ajout ou modification de tests
- **chore**: Autres changements (dépendances, etc.)
- **ci**: Changements des configurations CI/CD

## Breaking Changes

Pour un breaking change (génère un release majeur), ajoutez `BREAKING CHANGE:` au pied du commit :

```
feat(api): restructure response format

BREAKING CHANGE: the response format has changed
```

## Exemples

### Nouvelle fonctionnalité
```
feat(controller): add support for CloudflareRuleset

Add support for creating and managing Cloudflare rulesets
through Kubernetes resources.
```

### Correctif
```
fix(logger): correct log level handling

The log level was not being correctly applied from
the environment variable.
```

### Breaking change
```
feat(types): change CloudflareRecord spec format

Replace zoneId + name with domain + name structure.

BREAKING CHANGE: CloudflareRecord spec now requires
'domain' and 'name' instead of 'zoneId' and full name
```

## Automatisation

- ✅ Semantic Release automatise la versioning basée sur les commits
- ✅ CHANGELOG.md est généré automatiquement
- ✅ Releases GitHub sont créées automatiquement
- ✅ Les tags de versions sont créés automatiquement
- ✅ Les images Docker sont taggées avec les versions

## Validation des commits

Les commits sont validés avec commitlint. Pour vous aider, utilisez :

```bash
npm install -g commitizen
git cz
```

Ou utilisez le hook pre-commit fourni.
