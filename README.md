# Sbrigo

Sbrigo è una PWA per la gestione condivisa delle liste e delle attività di casa. La prima fase
sostituisce Google Keep per la spesa: una lista a spunta rapida, raggruppata per reparto e
ordinata secondo le corsie del supermercato scelto. Schema dati, API e interfaccia sono già
predisposti per le attività generali (scadenze, note, assegnazione ai membri della famiglia).

Il documento di requisiti e architettura è in [`docs/REQUISITI_E_ARCHITETTURA.md`](docs/REQUISITI_E_ARCHITETTURA.md),
il riferimento delle API in [`docs/API.md`](docs/API.md).

## Stack

| Componente      | Tecnologia                                                        |
| --------------- | ----------------------------------------------------------------- |
| Backend         | Go 1.24, `net/http`, pgx v5, go-oidc                              |
| Frontend        | Vue 3, TypeScript 6, Vite, Pinia, vue-router, Reka UI, vue-i18n (it/en), vite-plugin-pwa, IndexedDB (idb) |
| Database        | PostgreSQL (istanza esistente dell'homelab)                       |
| Autenticazione  | Logto via OIDC, Authorization Code + PKCE, sessioni in Redis      |
| Realtime        | Server-Sent Events                                                |
| Deploy          | Un solo container Docker (frontend embedded nel binario Go)       |

## Struttura del repository

```
cmd/sbrigo/            entrypoint del server
internal/api/          router, handler REST, SSE, serving della PWA
internal/auth/         OIDC + PKCE, cookie di sessione firmati, API key dell'agente
internal/config/       configurazione da variabili d'ambiente
internal/db/           pool PostgreSQL e migrazioni SQL embedded
internal/model/        tipi di dominio
internal/realtime/     broker SSE in memoria
internal/store/        accesso ai dati (users, departments, supermarkets, tasks)
web/                   PWA Vue (web/dist viene embedded nel binario)
docs/                  requisiti, architettura e API
```

## Funzionamento

- **Offline-first.** Il frontend salva tutto in IndexedDB e applica ogni modifica subito in
  locale. Le modifiche finiscono in una coda (outbox) che viene inviata al backend appena la
  connessione è disponibile. Le creazioni usano UUID generati dal client, quindi la coda è
  riproducibile in modo idempotente.
- **Last-write-wins.** Ogni modifica porta il timestamp del client in `updated_at`. Il server
  applica l'aggiornamento solo se non ne ha già uno più recente; in caso contrario risponde
  `409` con la riga corrente e il client la adotta.
- **Realtime.** Ogni scrittura viene pubblicata su `/api/v1/events` (SSE). I client connessi
  aggiornano la lista istantaneamente; a ogni riconnessione fanno un refresh completo.
- **Sessioni.** Il cookie contiene solo un token opaco; la sessione vive in Redis (indicizzata per
  utente, salvata come hash del token) e può essere revocata: `POST /auth/logout` chiude quella
  corrente, `POST /auth/logout-all` tutte quelle dell'utente. Senza Redis il backend ripiega su un
  cookie firmato HMAC, che non è revocabile prima della scadenza.
- **Utenti.** Al primo login il backend crea il profilo locale (`users`) a partire dai claim
  OIDC, così le attività possono essere assegnate ai membri della famiglia. Tutti gli utenti
  autenticati hanno permessi di lettura e scrittura completi.
- **Liste.** Spesa e attività sono composte da più liste (es. "Spesa", "Farmacia", "Casa",
  "Burocrazia"). Ogni lista appare come card con l'anteprima: elementi da fare, in ritardo e
  prossima scadenza, calcolati in locale così funzionano anche offline. La lista selezionata è
  ricordata per dispositivo.
- **Interfaccia.** Componenti accessibili di Reka UI (select, dialog a foglio, checkbox, toggle,
  toast). Testi in italiano e inglese con vue-i18n: la lingua segue il browser ed è cambiabile
  dalle Impostazioni, come il tema (sistema, chiaro, scuro). I colori sono token CSS in
  `web/src/styles.css`, un blocco per il chiaro e uno per lo scuro.
- **Agente.** Le stesse API sono utilizzabili con una API key statica nell'header `X-API-Key`.
  L'eventuale esposizione via MCP è demandata a un microservizio separato che consuma queste API.

## Configurazione

Tutte le variabili sono documentate in [`.env.example`](.env.example). Le principali:

| Variabile                    | Descrizione                                                       |
| ---------------------------- | ----------------------------------------------------------------- |
| `SBRIGO_DATABASE_URL`        | connection string PostgreSQL (obbligatoria)                       |
| `SBRIGO_SESSION_SECRET`      | chiave HMAC del cookie di login e del fallback stateless          |
| `SBRIGO_REDIS_URL`           | Redis per le sessioni lato server; vuoto = cookie firmati         |
| `SBRIGO_PUBLIC_URL`          | URL pubblico dietro il reverse proxy                              |
| `SBRIGO_OIDC_ISSUER`         | issuer Logto, es. `https://logto.casa.example/oidc`               |
| `SBRIGO_OIDC_CLIENT_ID`      | client id dell'applicazione Logto                                 |
| `SBRIGO_OIDC_CLIENT_SECRET`  | opzionale, solo per app "Traditional web"                         |
| `SBRIGO_API_KEY`             | API key dell'agente; vuota = disabilitata                         |
| `SBRIGO_DEV_AUTO_LOGIN_EMAIL`| solo sviluppo: autentica tutto come questo utente                 |

In Logto registrare come redirect URI `${SBRIGO_PUBLIC_URL}/auth/callback`.

## Sviluppo locale

Prerequisiti: Go 1.24, Node 22, un PostgreSQL raggiungibile.

```sh
# Backend (senza Logto, con utente di sviluppo)
SBRIGO_DATABASE_URL=postgres://user:pass@localhost:5432/sbrigo?sslmode=disable \
SBRIGO_SESSION_SECRET=una-stringa-casuale-di-almeno-32-caratteri \
SBRIGO_DEV_AUTO_LOGIN_EMAIL=dev@example.com \
SBRIGO_API_KEY=dev-key \
go run ./cmd/sbrigo

# Frontend con hot reload (proxy verso :8080)
cd web && npm install && npm run dev
```

Le migrazioni vengono applicate automaticamente all'avvio. Il seed iniziale crea sei reparti
con la relativa legenda, modificabili dalla schermata Impostazioni.

### Test

```sh
go test ./...                                         # unit test Go
SBRIGO_TEST_DATABASE_URL=postgres://... go test ./... # anche i test di integrazione (schema ricreato!)
golangci-lint run                                     # lint Go (config in .golangci.yml)
cd web && npm run lint && npm test && npm run build   # ESLint + Prettier, Vitest, typecheck e build PWA
```

Il database indicato in `SBRIGO_TEST_DATABASE_URL` viene svuotato a ogni test: usarne uno dedicato.

## Deploy

```sh
docker build -t sbrigo .
docker run -d --name sbrigo -p 8080:8080 --env-file .env sbrigo
```

Oppure con `docker compose up -d` (il compose incluso avvia anche un PostgreSQL di comodo; in
homelab basta rimuovere il servizio `db` e puntare `SBRIGO_DATABASE_URL` all'istanza esistente).

Dietro il reverse proxy vanno disabilitati buffering e timeout di lettura sul path
`/api/v1/events` (SSE). Con nginx: `proxy_buffering off; proxy_read_timeout 1h;`. L'endpoint
`/healthz` verifica anche la raggiungibilità del database.
