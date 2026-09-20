# Documento di Requisiti e Architettura: Sbrigo

## 1. Visione ed evoluzione del progetto
Sbrigo è un'applicazione web progressiva progettata per la gestione centralizzata delle liste e delle attività del nucleo familiare. L'obiettivo iniziale è fornire un'interfaccia immediata per la spesa al supermercato, sostituendo Google Keep con una lista a spunta rapida e ordinata per reparti.

L'architettura del sistema e lo schema del database sono strutturati fin da subito per supportare l'evoluzione verso una gestione completa delle attività di casa. Nelle fasi successive l'interfaccia accoglierà task complessi con scadenze, note articolate e assegnazioni ai vari membri della famiglia, unificando spesa e burocrazia in un solo strumento.

## 2. Architettura tecnica e stack
Il sistema è composto da un unico container Docker distribuito sull'infrastruttura di casa dietro reverse proxy.

Il backend è realizzato in Go, garantendo tempi di risposta rapidi e un consumo di risorse ridotto. Il frontend è sviluppato in Vue.js e configurato come PWA installabile su smartphone. La persistenza dei dati si appoggia all'istanza PostgreSQL già presente nell'homelab.

## 3. Autenticazione e gestione utenti
L'accesso a Sbrigo è protetto dall'identity provider Logto via OIDC con flusso Authorization Code e PKCE.

Al primo accesso di un utente, il backend sincronizza il profilo locale creando un record nella tabella degli utenti. Questo passaggio è indispensabile per permettere l'assegnazione delle attività ai singoli componenti della famiglia. Tutti gli utenti autenticati dal provider hanno permessi di lettura e scrittura su tutta la piattaforma.

## 4. Sincronizzazione in tempo reale e funzionamento offline
Gli aggiornamenti tra dispositivi avvengono tramite Server-Sent Events. Ogni azione eseguita su un telefono viene inviata al server e notificata istantaneamente agli altri client connessi.

L'applicazione adotta un approccio offline-first tramite Service Worker e IndexedDB. Tutte le modifiche vengono applicate subito nell'interfaccia locale e inserite in una coda temporanea. Non appena la connessione internet torna disponibile, Sbrigo invia la coda al backend. In caso di modifiche sovrapposte, il server applica la regola dell'ultimo aggiornamento ricevuto in base al timestamp.

## 5. Interfaccia utente e modalità di visualizzazione
L'interfaccia principale per la spesa è progettata per l'uso a una mano: presenta una sequenza verticale di elementi con caselle di spunta, raggruppati sotto l'intestazione del rispettivo reparto con la relativa legenda visibile.

Un selettore in alto permette di scegliere il supermercato corrente, riorganizzando l'ordine delle sezioni in base alla sequenza reale delle corsie.

L'interfaccia include la predisposizione per passare a viste dedicate alle attività generali, dove per ogni elemento sarà possibile visualizzare e modificare la data di scadenza, il membro della famiglia assegnato e le note di dettaglio.

## 6. API REST e integrazione agente
Il backend Go espone una serie di API REST protette tramite API Key statica passata nell'header HTTP.

Le API consentono all'agente di leggere, creare, aggiornare, spuntare o riassegnare qualsiasi elemento del sistema, sia esso un articolo della spesa o una pratica burocratica. L'eventualità di esporre queste funzionalità tramite protocollo MCP viene gestita tramite un microservizio integrativo separato.

## 7. Schema del database (PostgreSQL)

### Tabella users
- id (UUID, Primary Key)
- identity_id (VARCHAR, unique, ID proveniente da Logto)
- email (VARCHAR)
- display_name (VARCHAR)
- created_at (TIMESTAMP WITH TIME ZONE)

### Tabella departments
- id (UUID, Primary Key)
- name (VARCHAR, es. "Dispensa", "Frigo", "Altro")
- description (TEXT, legenda esplicativa visibile nell'interfaccia)
- default_sort_order (INTEGER)

### Tabella supermarkets
- id (UUID, Primary Key)
- name (VARCHAR, es. "Esselunga", "Lidl")

### Tabella supermarket_department_orders
- supermarket_id (UUID, Foreign Key)
- department_id (UUID, Foreign Key)
- sort_order (INTEGER)
- Primary Key composta: (supermarket_id, department_id)

### Tabella tasks
- id (UUID, Primary Key)
- title (VARCHAR)
- notes (TEXT, nullable)
- is_completed (BOOLEAN, default false)
- item_type (VARCHAR, es. "grocery" oppure "general_task")
- department_id (UUID, Foreign Key, nullable)
- assignee_id (UUID, Foreign Key su users, nullable)
- due_date (TIMESTAMP WITH TIME ZONE, nullable)
- created_at (TIMESTAMP WITH TIME ZONE)
- updated_at (TIMESTAMP WITH TIME ZONE)
