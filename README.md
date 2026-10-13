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

```bash
docker compose up --build
```

Ouvrez ensuite `http://localhost:5173`. L'API est disponible sur
`http://localhost:8080` et MongoDB sur `localhost:27017`.

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

Le dossier `server/` contient un serveur API minimal en mémoire, utilisable comme base
pour brancher une base de données et une vraie authentification.

Dans un premier terminal :

```bash
cd server
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

Le backend utilise maintenant MongoDB. Exportez ces variables ou créez un fichier
`server/.env` local (non versionné). Le serveur charge automatiquement ce fichier
`.env` en développement :

```bash
export MONGODB_URI=mongodb://localhost:27017
export MONGODB_DATABASE=siem
export PORT=8080
export CORS_ORIGIN=http://localhost:5173
```

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

## Configuration

Copiez `.env.example` en `.env` et ajustez :

| Variable | Description | Défaut |
|----------|-------------|--------|
| `VITE_API_URL` | URL de base de l'API REST | `http://localhost:8080` |

## Routes

| Route | Description | Accès |
|-------|-------------|-------|
| `/login` | Connexion | Public |
| `/signup` | Création de compte entreprise (2 étapes) | Public |
| `/invite/:token` | Acceptation d'invitation | Public |
| `/` | Dashboard | Authentifié |
| `/events` | Journal des événements | Authentifié |
| `/alerts` | Alertes de détection | Authentifié |
| `/addresses` | Adresses surveillées | Authentifié |
| `/team` | Gestion de l'équipe | owner, admin |

## Authentification & rôles

L'application nécessite une authentification. Chaque utilisateur appartient à une entreprise (company) et a un rôle : `owner`, `admin`, ou `viewer`.

- Le token est stocké dans `sessionStorage` et la session est restaurée au rechargement via `GET /api/auth/me`.
- La page de login affiche un message générique "Invalid credentials" pour ne pas révéler quels emails existent.
- Les viewers n'ont pas accès à `/team` (redirection + toast).
- Le propriétaire (`owner`) ne peut pas être supprimé ou rétrogradé.
- Les admins ne peuvent pas modifier le propriétaire ou d'autres admins.

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
3. Le frontend appelle les endpoints suivants :

### Authentification

| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/api/auth/login` | POST | Connexion `{ email, password }` → `{ token, user }` |
| `/api/auth/me` | GET | Récupère l'utilisateur courant à partir du token |
| `/api/auth/signup` | POST | Création de compte `{ company: {...}, admin: {...} }` → `{ token, user }` |

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

Tous les endpoints stats/events/alerts acceptent un paramètre optionnel `address=` pour filtrer par adresse surveillée. Toutes les réponses doivent inclure les en-têtes CORS appropriés. Le header `Authorization: Bearer <token>` est ajouté automatiquement à chaque requête. Un 401 déconnecte automatiquement l'utilisateur. Les erreurs API doivent renvoyer un champ `error` ou `message` dans le body JSON pour un message lisible.

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
