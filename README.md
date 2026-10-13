# Mini SIEM Dashboard

Dashboard frontend "Security Information & Event Management" — interface SOC / cybersécurité en Vue 3 + TypeScript.

## Stack

- **Vue 3** (Composition API, `<script setup lang="ts">`)
- **TypeScript** strict, **Vite**
- **Vue Router** + **Pinia**
- **Chart.js** + **vue-chartjs** (graphiques)
- **CSS pur** (variables CSS, flexbox, grid) — aucune librairie UI

## Démarrage

```bash
npm install
npm run dev
```

L'application utilise l'API Go et MongoDB comme source de données.

### Lancer tout le projet avec Docker

Docker Compose démarre MongoDB, l'API Go et le frontend Vite. Les dossiers
du frontend et du backend sont montés dans les conteneurs afin que les
modifications du code soient prises en compte sans reconstruire les images.

Le seul secret obligatoire est `JWT_SECRET`, lu depuis le fichier `.env` à la
racine. Sans lui, `docker compose` refuse de démarrer :

```bash
cp .env.example .env
echo "JWT_SECRET=$(openssl rand -hex 32)" >> .env
docker compose up --build
```

Ouvrez ensuite `http://localhost:5173`. L'API est disponible sur
`http://localhost:8080` et MongoDB sur `127.0.0.1:27017`. MongoDB et Mailpit
n'écoutent que sur `127.0.0.1`, car ils n'ont pas d'authentification.

Les emails (invitations, réinitialisation de mot de passe) sont capturés en
local par **Mailpit** : ouvrez la boîte de réception sur `http://localhost:8025`.
Aucun email ne quitte votre machine en développement.

Arrêter les services :

```bash
docker compose down
```

Supprimer aussi les données MongoDB :

```bash
docker compose down -v
```

La migration MongoDB est exécutée automatiquement par le conteneur MongoDB
lors de sa première initialisation.

### Lancer le serveur Go

Le dossier `server/` contient l'API Go (MongoDB, JWT, envoi d'emails).

Dans un premier terminal (MongoDB doit tourner, par exemple
`docker compose up -d mongo mailpit`) :

```bash
cd server
cp .env.example .env
sed -i "s/^JWT_SECRET=$/JWT_SECRET=$(openssl rand -hex 32)/" .env
go run ./cmd/server
```

Dans un second terminal, à la racine du projet :

```bash
cp .env.example .env
npm install
npm run dev
```

Le serveur écoute sur `http://localhost:8080` et l'interface Vite sur
`http://localhost:5173`. Vérifiez l'API avec `curl http://localhost:8080/healthz`.

### Structure du backend Go

```text
server/
├── cmd/server/                # Point d'entrée de l'application
├── internal/config/           # Configuration par variables d'environnement
├── internal/api/              # Contrôleurs et routes HTTP
├── internal/middleware/       # Middleware HTTP
├── internal/models/           # Modèles et payloads JSON
├── internal/repository/       # Accès aux données MongoDB
└── migrations/                # Migrations MongoDB
```

Le backend lit sa configuration dans l'environnement ou dans `server/.env`
(non versionné, modèle : `server/.env.example`). Toutes les variables sont
décrites dans la section [Configuration](#configuration). Le serveur refuse de
démarrer avec un message clair si la configuration est invalide (secret
manquant, durée mal formée, `*` dans CORS, cookie non sécurisé en production…).

Pour préparer MongoDB :

```bash
mongosh "$MONGODB_URI/siem" server/migrations/001_initial.js
```

La migration crée les collections `companies`, `users`, `addresses`, `events`,
`alerts` et `invitations`, ainsi que les index nécessaires au filtrage par
entreprise, sévérité, IP, statut et date.

Le découpage suit une architecture MVC adaptée à une API :

- **Models** : structures JSON/BSON dans `internal/models`
- **Controllers** : handlers HTTP dans `internal/api`
- **Repository** : accès MongoDB dans `internal/repository`
- **Configuration** : environnement dans `internal/config`
- **Routes et middleware** : assemblés dans `internal/api` et `internal/middleware`

## Tests

Frontend (Vitest + Vue Test Utils, jsdom) :

```bash
npm test            # une exécution
npm run test:watch  # mode watch
```

Backend (Go). Les tests d'intégration passent par les vraies routes HTTP
(JWT, handlers, MongoDB) et ne s'exécutent que si `MONGODB_TEST_URI` est
défini ; sans lui ils sont ignorés (`SKIP`) :

```bash
cd server
go test ./...                                   # tests unitaires seuls
docker run -d --rm --name siem-test-mongo --ulimit nofile=64000:64000 -p 127.0.0.1:27018:27017 mongo:7
MONGODB_TEST_URI=mongodb://localhost:27018 go test -race ./...
```

Chaque test crée puis supprime sa propre base `siem_test_*`. La CI GitHub
Actions (`.github/workflows/ci.yml`) lance build, vet et tests backend avec un
service MongoDB, puis build et tests frontend, à chaque push.

## Configuration

Frontend et Docker Compose : `.env` à la racine (modèle `.env.example`).

| Variable | Description | Défaut |
|----------|-------------|--------|
| `VITE_API_URL` | URL de base de l'API REST | `http://localhost:8080` |
| `JWT_SECRET` | Lu par Docker Compose pour l'API (jamais exposé au navigateur) | **obligatoire** |

Backend : environnement ou `server/.env` (modèle `server/.env.example`).

| Variable | Description | Défaut |
|----------|-------------|--------|
| `APP_ENV` | `development`, `production` ou `test`. En `development` uniquement, les emails non envoyés (SMTP absent) sont écrits en clair dans les logs | `production` |
| `PORT` | Port HTTP de l'API | `8080` |
| `MONGODB_URI` / `MONGODB_DATABASE` | Connexion MongoDB | `mongodb://localhost:27017` / `siem` |
| `MONGO_CONNECT_RETRIES` | Tentatives de connexion au démarrage | `12` |
| `CORS_ALLOWED_ORIGINS` | Origines autorisées, séparées par des virgules (`*` refusé) | `http://localhost:5173` |
| `JWT_SECRET` | Secret HS256, au moins 32 caractères (`openssl rand -hex 32`) | **obligatoire** |
| `JWT_TTL` | Durée de vie du jeton d'accès | `15m` |
| `REFRESH_TTL` | Durée de vie du refresh token (doit dépasser `JWT_TTL`) | `168h` |
| `COOKIE_SECURE` | Cookie de refresh `Secure` ; `false` seulement en http local, refusé en production | `true` |
| `TRUST_PROXY` | Faire confiance à `X-Forwarded-For` (uniquement derrière un reverse proxy) | `false` |
| `RATE_LIMIT_WINDOW` | Fenêtre des limites de débit | `1m` |
| `RATE_LIMIT_AUTH_PER_IP` | Requêtes d'authentification par IP et par route dans la fenêtre | `20` |
| `RATE_LIMIT_AUTH_PER_EMAIL` | Tentatives par email (ou par jeton de reset) dans la fenêtre | `5` |
| `APP_BASE_URL` | URL publique du frontend pour les liens des emails (https en production) | `http://localhost:5173` |
| `SMTP_HOST` | Serveur SMTP ; vide = emails seulement journalisés | vide |
| `SMTP_PORT` | Port SMTP | `587` |
| `SMTP_USER` / `SMTP_PASSWORD` | Identifiants SMTP (les deux ou aucun) | vide |
| `SMTP_REQUIRE_TLS` | STARTTLS obligatoire ; `false` seulement pour Mailpit, refusé en production | `true` |
| `MAIL_FROM` | Expéditeur, obligatoire si `SMTP_HOST` est défini | vide |

## Routes

| Route | Description | Accès |
|-------|-------------|-------|
| `/login` | Connexion | Public |
| `/signup` | Création de compte entreprise (2 étapes) | Public |
| `/invite/:token` | Acceptation d'invitation | Public |
| `/forgot-password` | Demande de réinitialisation du mot de passe | Public |
| `/reset-password/:token` | Choix d'un nouveau mot de passe | Public |
| `/` | Dashboard | Authentifié |
| `/events` | Journal des événements | Authentifié |
| `/alerts` | Alertes de détection | Authentifié |
| `/addresses` | Adresses surveillées | Authentifié |
| `/team` | Gestion de l'équipe | owner, admin |

## Authentification & rôles

L'application nécessite une authentification. Chaque utilisateur appartient à une entreprise (company) et a un rôle : `owner`, `admin`, ou `viewer`.

- Le token est stocké dans `sessionStorage` et la session est restaurée au rechargement via `GET /api/auth/me`.
- Voir [Flux d'authentification](#flux-dauthentification) pour les JWT, le refresh et la réinitialisation.
- La page de login affiche un message générique "Invalid credentials" pour ne pas révéler quels emails existent.
- Les viewers n'ont pas accès à `/team` (redirection + toast).
- Le propriétaire (`owner`) ne peut pas être supprimé ou rétrogradé.
- Les admins ne peuvent pas modifier le propriétaire ou d'autres admins.

### Flux d'authentification

1. **Connexion / inscription / invitation acceptée** : l'API renvoie un **jeton
   d'accès JWT** (HS256, 15 min par défaut, avec l'id utilisateur, l'entreprise,
   le rôle et une `tokenVersion`) et pose un **refresh token** dans un cookie
   `siem_refresh` `HttpOnly`, `SameSite=Strict`, `Secure` (hors dev),
   limité au chemin `/api/auth`. Le JWT est gardé dans `sessionStorage` ; le
   refresh token n'est jamais lisible par le JavaScript.
2. **Requêtes** : `Authorization: Bearer <jwt>`. Le middleware vérifie la
   signature, l'algorithme (HS256 uniquement), l'expiration et la
   `tokenVersion` de l'utilisateur.
3. **Refresh** : sur un 401, le client appelle une seule fois
   `POST /api/auth/refresh`. Le refresh token est à usage unique (rotation à
   chaque appel) ; réutiliser un ancien jeton révoque toutes les sessions de
   l'utilisateur. Si le refresh échoue, les stores sont vidés et l'utilisateur
   est renvoyé vers `/login`.
4. **Déconnexion** : `POST /api/auth/logout` révoque le refresh token côté
   serveur, efface le cookie et vide tous les stores Pinia.
5. **Révocation** : changer le rôle, révoquer un membre ou réinitialiser le
   mot de passe incrémente `tokenVersion`, ce qui invalide immédiatement tous
   les jetons d'accès et refresh existants.
6. **Mot de passe oublié** : `POST /api/auth/forgot-password` répond toujours
   le même message. Si le compte existe, un lien
   `APP_BASE_URL/reset-password/<token>` valable 30 minutes et à usage unique
   est envoyé par email. `POST /api/auth/reset-password` applique les règles de
   mot de passe de l'inscription, ferme toutes les sessions et envoie un email
   de confirmation.

Les mots de passe sont hachés en Argon2id. Seuls les hachages SHA-256 des
refresh tokens, jetons d'invitation (7 jours) et jetons de réinitialisation
sont stockés. Login, inscription, refresh, mot de passe oublié,
réinitialisation et invitations sont limités en débit (429 + `Retry-After`).

### Inscription

Utilisez `/signup` pour créer une entreprise et son premier compte administrateur.
Le premier utilisateur reçoit le rôle `owner`, est connecté automatiquement et
est redirigé vers le dashboard. La gestion des invitations reste disponible
depuis la page `/team`.

### Inscription entreprise (/signup)

Formulaire en 2 étapes avec indicateur de progression :
1. **Informations entreprise** : nom, secteur, taille, pays, site web (optionnel), contact sécurité
2. **Compte administrateur** : nom complet, email professionnel, mot de passe (avec indicateur de force et règles en temps réel), confirmation, acceptation des conditions

Le premier utilisateur d'une entreprise reçoit le rôle `owner`. Après
inscription, l'utilisateur est connecté automatiquement et redirigé vers `/`.

### Gestion de l'équipe (/team)

Visible uniquement par les `owner` et `admin` :
- Invitation individuelle par email avec choix du rôle (admin ou viewer)
- Ajout en masse : textarea pour coller plusieurs emails séparés par virgules, espaces ou sauts de ligne, avec aperçu des emails valides/invalides
- Tableau des membres : email, rôle (modifiable par le owner), statut (invited/active/revoked), date d'invitation, dernière connexion
- Actions : renvoyer une invitation, révoquer l'accès (dialogue de confirmation), supprimer un membre

### Acceptation d'invitation (/invite/:token)

- Affiche le nom de l'entreprise, l'email invité (en lecture seule) et le rôle
- Formulaire : nom complet + mot de passe avec indicateur de force
- Écrans d'erreur dédiés pour les invitations invalides, expirées ou déjà utilisées

## Brancher l'API Go réelle

1. Définissez `VITE_API_URL` avec l'URL de votre backend Go.
2. Le frontend appelle les endpoints suivants :

### Authentification

| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/api/auth/login` | POST | Connexion `{ email, password }` → `{ token, user }` |
| `/api/auth/me` | GET | Récupère l'utilisateur courant à partir du token |
| `/api/auth/signup` | POST | Création de compte `{ company: {...}, admin: {...} }` → `{ token, user }` |
| `/api/auth/refresh` | POST | Nouveau jeton d'accès à partir du cookie de refresh (rotation) → `{ token, user }` |
| `/api/auth/logout` | POST | Révoque le refresh token et efface le cookie |
| `/api/auth/forgot-password` | POST | `{ email }` → toujours 200 avec le même message |
| `/api/auth/reset-password` | POST | `{ token, newPassword }` → 200, ou 400 générique si le lien est invalide |

### Équipe

| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/api/team` | GET | Liste des membres de l'entreprise |
| `/api/team/invitations` | POST | Invitations en masse `{ emails: string[], role }` → `{ created, skipped }` |
| `/api/team/:id/resend` | POST | Renvoyer une invitation |
| `/api/team/:id` | PATCH | Modifier le rôle ou le statut `{ role?, status? }` |
| `/api/team/:id` | DELETE | Supprimer un membre |

### Invitations

| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/api/invitations/:token` | GET | Détails d'une invitation `{ companyName, email, role }` |
| `/api/invitations/:token/accept` | POST | Accepter `{ fullName, password }` → `{ token, user }` |

### Adresses surveillées

| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/api/addresses` | GET | Liste des adresses de l'entreprise |
| `/api/addresses` | POST | Ajoute une adresse `{ value, label? }` |
| `/api/addresses/:id` | DELETE | Supprime une adresse |

### Stats, events, alerts

| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/api/stats/summary` | GET | Statistiques globales |
| `/api/stats/timeline?range=1h\|24h\|7d` | GET | Série temporelle |
| `/api/stats/top-ips?limit=10` | GET | Top 10 IPs sources |
| `/api/events?page=&limit=&source=&severity=&ip=&q=&from=&to=&address=` | GET | Liste paginée d'événements |
| `/api/alerts?status=&address=` | GET | Liste des alertes |
| `/api/alerts/:id` | PATCH | Met à jour le statut d'une alerte |

Tous les endpoints stats/events/alerts acceptent un paramètre optionnel `address=` pour filtrer par adresse surveillée. Toutes les réponses doivent inclure les en-têtes CORS appropriés. Le header `Authorization: Bearer <token>` est ajouté automatiquement à chaque requête. Sur un 401, le client tente un refresh silencieux, puis déconnecte l'utilisateur si le refresh échoue. Les erreurs API doivent renvoyer un champ `error` ou `message` dans le body JSON pour un message lisible.

### Types attendus

```typescript
Role = 'owner' | 'admin' | 'viewer'
User = { id, email, fullName?, companyId, companyName, role: Role }
TeamMember = { id, email, fullName?, role: Role, status: 'invited' | 'active' | 'revoked', invitedAt, lastLoginAt? }
Address = { id, value, label?, createdAt }
Event = { id, timestamp, source, severity, ip, message, rawLog }
Alert = { id, createdAt, rule, severity, ip, description, eventCount, status }
```

## Structure du projet

```
src/
├── api/           # Couche HTTP — une fonction typée par endpoint, auth + 401
├── types/         # Types TypeScript partagés
├── stores/        # Stores Pinia (auth, dashboard, events, alerts, addresses, toast)
├── composables/   # useCan() — vérifications de rôles pour masquer/afficher des actions
├── utils/         # Validators — fonctions pures testables (email, password, domain, etc.)
├── components/    # Composants réutilisables (StatCard, SeverityBadge, charts, Toast, PasswordStrengthMeter, etc.)
├── views/         # Pages (Login, Signup, Invite, Dashboard, Events, Alerts, Addresses, Team)
├── router/        # Configuration Vue Router avec guards d'authentification et de rôles
└── assets/
    └── styles/    # CSS global (thème dark SOC)
```

## Fonctionnalités

- **Authentification** : login, signup 2 étapes, session restaurée au rechargement, route guards, redirection 401
- **Rôles** : owner, admin, viewer — guards sur les routes, `useCan()` pour masquer les actions
- **Équipe** : invitations individuelles et en masse, gestion des rôles, révocation, suppression, renvoi
- **Invitations** : page d'acceptation avec validation de token, écrans d'erreur dédiés
- **Multi-entreprise** : données isolées par entreprise, stores nettoyés à la déconnexion
- **Adresses surveillées** : page dédiée avec ajout/suppression, sélecteur global dans le header
- **Dashboard** : 4 cartes de stats, graphique temporel (1h/24h/7d), top 10 IPs, alertes critiques
- **Events** : tableau filtrable, pagination server-side, drawer de détail (fermable avec Escape)
- **Alerts** : cartes filtrables, acknowledge/resolve avec mise à jour optimiste et toast
- **Auto-refresh** : toggle sur le dashboard (10s)
- **États** : skeleton loaders, error + retry, empty states — sur chaque vue
- **Toasts** : notifications de succès/erreur
- **Responsive** : sidebar collapsible, tables scrollables, header adapté
- **Accessibilité** : labels sur tous les inputs, aria-live pour les erreurs, navigation clavier dans le formulaire à 2 étapes, focus sur le premier champ invalide, focus rings, contraste
