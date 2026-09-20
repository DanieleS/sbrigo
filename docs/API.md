# API REST di Sbrigo

Base path: `/api/v1`. Tutte le richieste e le risposte sono JSON. Timestamp in RFC 3339 (UTC).

## Autenticazione

Due modalità, valide su tutti gli endpoint:

| Modalità          | Come                                                                  |
| ----------------- | --------------------------------------------------------------------- |
| Utente (browser)  | cookie di sessione `sbrigo_session` rilasciato dal flusso OIDC        |
| Agente            | header `X-API-Key: <SBRIGO_API_KEY>`                                  |

Senza credenziali valide la risposta è `401 {"error":"authentication required"}`.

Flusso di login: `GET /auth/login[?return_to=/percorso]` → redirect a Logto → `GET /auth/callback`
→ cookie di sessione e redirect a `return_to`. Logout: `POST /auth/logout` (204) chiude la sessione
corrente; `POST /auth/logout-all` (204) revoca tutte le sessioni dell'utente (richiede Redis, altrimenti 501).

## Formato degli errori

```json
{ "error": "descrizione" }
```

`400` input non valido, `401` non autenticato, `404` risorsa inesistente, `409` conflitto
last-write-wins (il body è la riga corrente, non un errore), `500` errore interno.

## Endpoint

### Utenti

| Metodo | Path        | Descrizione                                                     |
| ------ | ----------- | --------------------------------------------------------------- |
| GET    | `/me`       | `{ "kind": "user"\|"agent", "user": User\|null }`               |
| GET    | `/users`    | elenco dei membri della famiglia (per l'assegnazione)           |

### Reparti

| Metodo | Path                 | Body                                                  |
| ------ | -------------------- | ----------------------------------------------------- |
| GET    | `/departments`       | —                                                     |
| POST   | `/departments`       | `{ "name", "description", "default_sort_order" }`     |
| PUT    | `/departments/{id}`  | `{ "name", "description", "default_sort_order" }`     |
| DELETE | `/departments/{id}`  | — (gli articoli collegati restano senza reparto)      |

### Supermercati e ordine delle corsie

| Metodo | Path                                    | Body                                   |
| ------ | --------------------------------------- | -------------------------------------- |
| GET    | `/supermarkets`                         | —                                      |
| POST   | `/supermarkets`                         | `{ "name" }`                           |
| PUT    | `/supermarkets/{id}`                    | `{ "name" }`                           |
| DELETE | `/supermarkets/{id}`                    | —                                      |
| GET    | `/supermarkets/{id}/department-order`   | → `[{ "department_id", "sort_order" }]`|
| PUT    | `/supermarkets/{id}/department-order`   | `{ "department_ids": [uuid, ...] }`    |

`PUT department-order` sostituisce l'intera sequenza. I reparti non elencati vengono mostrati
dopo quelli ordinati, secondo `default_sort_order`.

### Liste

Ogni task appartiene a una lista. Il `kind` della lista (`grocery` o `general_task`) determina il
tipo dei suoi elementi ed è immutabile. Deve esistere sempre almeno una lista per tipo.

| Metodo | Path           | Body / note                                                              |
| ------ | -------------- | ------------------------------------------------------------------------ |
| GET    | `/lists`       | elenco con anteprima: `open_count`, `done_count`, `overdue_count`, `next_due` |
| POST   | `/lists`       | `{ "name", "kind", "sort_order" }`                                       |
| PUT    | `/lists/{id}`  | `{ "name", "sort_order" }`                                               |
| DELETE | `/lists/{id}`  | elimina la lista **e i suoi elementi**; `409` se è l'ultima del suo tipo |

### Task (articoli della spesa e attività)

Oggetto `Task`:

```json
{
  "id": "uuid",
  "list_id": "uuid",
  "title": "Latte",
  "notes": null,
  "is_completed": false,
  "item_type": "grocery",          // oppure "general_task"
  "department_id": "uuid|null",
  "assignee_id": "uuid|null",
  "due_date": "2026-09-25T10:00:00Z|null",
  "created_at": "...",
  "updated_at": "..."
}
```

| Metodo | Path                              | Descrizione                                                    |
| ------ | --------------------------------- | -------------------------------------------------------------- |
| GET    | `/tasks`                          | filtri: `list_id`, `type`, `completed`, `department_id`, `assignee_id`, `since` |
| POST   | `/tasks`                          | crea; senza `list_id` va nella prima lista del tipo (`item_type` default `grocery`); con `list_id` il tipo segue la lista |
| GET    | `/tasks/{id}`                     | dettaglio                                                      |
| PATCH  | `/tasks/{id}`                     | aggiornamento parziale                                         |
| DELETE | `/tasks/{id}`                     | elimina                                                        |
| DELETE | `/tasks?completed=true[&type=…][&list_id=…]` | elimina i completati → `{ "deleted": n, "ids": [...] }` |

**Creazione idempotente.** `POST /tasks` accetta opzionalmente `id`, `created_at` e `updated_at`
forniti dal client. Se l'`id` esiste già la risposta è `200` con la riga esistente (nessuna
modifica); altrimenti `201`. Questo permette di riprodurre la coda offline senza duplicati.

**Spostare tra liste.** `PATCH` con `list_id` sposta l'elemento; se la lista di destinazione è di
un altro tipo, `item_type` cambia di conseguenza. Un `PATCH` con solo `item_type` sposta
l'elemento nella prima lista di quel tipo.

**PATCH e last-write-wins.** Nel body vanno solo i campi da cambiare; `null` esplicito azzera un
campo nullable (`notes`, `department_id`, `assignee_id`, `due_date`). Il campo opzionale
`updated_at` è il timestamp della modifica sul client: il server la applica solo se non ha già
una versione più recente, altrimenti risponde `409` con la riga corrente. Senza `updated_at`
viene usato l'orologio del server (uso tipico dell'agente).

Esempi per l'agente:

```sh
# Aggiungere un articolo alla spesa nel reparto Frigo
curl -X POST https://sbrigo.casa.example/api/v1/tasks \
  -H "X-API-Key: $KEY" -H "Content-Type: application/json" \
  -d '{"title":"Latte","department_id":"<uuid reparto>"}'

# Spuntare un articolo
curl -X PATCH https://sbrigo.casa.example/api/v1/tasks/<id> \
  -H "X-API-Key: $KEY" -H "Content-Type: application/json" \
  -d '{"is_completed":true}'

# Creare una pratica assegnata a un familiare con scadenza
curl -X POST https://sbrigo.casa.example/api/v1/tasks \
  -H "X-API-Key: $KEY" -H "Content-Type: application/json" \
  -d '{"title":"Pagare bollo auto","item_type":"general_task","assignee_id":"<uuid utente>","due_date":"2026-10-31T12:00:00Z","notes":"Scade a fine mese"}'

# Riassegnare
curl -X PATCH https://sbrigo.casa.example/api/v1/tasks/<id> \
  -H "X-API-Key: $KEY" -H "Content-Type: application/json" \
  -d '{"assignee_id":"<uuid altro utente>"}'
```

### Eventi realtime

`GET /api/v1/events` apre uno stream Server-Sent Events. All'apertura viene inviato `event: connected`;
poi, a ogni scrittura, un evento con `id` progressivo, `event` fra:

`task.created`, `task.updated` (data: il `Task`), `task.deleted` (data: `{ "id" }`),
`department.created|updated|deleted`, `supermarket.created|updated|deleted`, `list.created|updated|deleted`,
`supermarket.order_updated` (data: `{ "supermarket_id", "order": [...] }`).

Ogni 25 secondi viene inviato un commento `: ping` per tenere viva la connessione attraverso il
reverse proxy. Non c'è replay degli eventi persi: alla riconnessione il client ricarica lo stato.

### Salute

`GET /healthz` → `200 ok` se il database risponde, `503` altrimenti. Non richiede autenticazione.
