# Sélection de Dix Projets Open-Source en Pure Go (`CGO_ENABLED=0`)

Document de référence répertoriant dix projets système, réseau bas niveau et embarqué développés intégralement en Go pur sans dépendance CGO. Ces projets constituent des cibles privilégiées pour des bancs d'essai de performance reproductibles, l'optimisation mémoire et l'apport de contributions d'ingénierie ciblées.

---

## Catalogue des Dix Projets Système & Embarqué

### 1. `tinygo-org/drivers`
* **Dépôt :** [`github.com/tinygo-org/drivers`](https://github.com/tinygo-org/drivers)
* **Domaine :** Système embarqué & contrôle matériel.
* **Description :** Collection de pilotes de périphériques (capteurs, afficheurs, bus I2C/SPI/UART) rédigés à 100 % en Go pur pour microcontrôleurs (ARM Cortex-M, RISC-V, ESP32).
* **Axe d'optimisation & Benchmarks :** Empreinte mémoire sur le tas (*heap allocations*), zéro-allocation sous interruptions matérielles, réduction du binaire et performances d'accès aux registres I/O.

### 2. `pion/webrtc`
* **Dépôt :** [`github.com/pion/webrtc`](https://github.com/pion/webrtc)
* **Domaine :** Pile réseau et multimédia temps réel.
* **Description :** Implémentation complète et autonome du protocole WebRTC (ICE, DTLS, SRTP, SCTP) à 100 % en Pure Go sans s'appuyer sur la bibliothèque C `libwebrtc`.
* **Axe d'optimisation & Benchmarks :** Débit d'I/O réseau, réutilisation des tampons mémoire (`sync.Pool`), élimination des copies dans la pile de paquets UDP et contrôle de latence sous forte charge.

### 3. `cockroachdb/pebble`
* **Dépôt :** [`github.com/cockroachdb/pebble`](https://github.com/cockroachdb/pebble)
* **Domaine :** Moteur de stockage clé-valeur (*Storage Engine* LSM-tree).
* **Description :** Moteur de stockage écrit en Go pur développé par CockroachDB pour remplacer RocksDB (qui imposait une dépendance CGO lourde), garantissant la gestion déterministe de téraoctets de données.
* **Axe d'optimisation & Benchmarks :** Bancs d'essai intensifs sur le compactage des fichiers SSTable, le contrôle du *write stall*, l'alignement des tampons d'écritures disques et la gestion de la mémoire sous concurrence.

### 4. `etcd-io/bbolt`
* **Dépôt :** [`github.com/etcd-io/bbolt`](https://github.com/etcd-io/bbolt)
* **Domaine :** Base de données embarquée B+Tree.
* **Description :** Fork maintenu par la CNCF et le projet `etcd` de la base de données BoltDB. Implémentation Go pur s'appuyant sur la projection mémoire (`mmap`).
* **Axe d'optimisation & Benchmarks :** Latence des parcours de nœuds B+Tree, zéro-copie sur la lecture de pages disques, verrouillage concurrent et coût des transactions en lecture seule.

### 5. `panjf2000/gnet`
* **Domaine :** Moteur réseau événementiel (*Event Loop* non-bloquant).
* **Dépôt :** [`github.com/panjf2000/gnet`](https://github.com/panjf2000/gnet)
* **Description :** Framework réseau bas niveau s'appuyant sur les primitives système `epoll` (Linux) et `kqueue` (BSD/macOS) en Go pur pour s'affranchir du modèle une-goroutine-par-connexion du paquet `net` standard.
* **Axe d'optimisation & Benchmarks :** Débit maximal en requêtes par seconde (RPS), réduction drastique des allocations sur le chemin critique des I/O et gestion fine de la mémoire réseau.

### 6. `quic-go/quic-go`
* **Domaine :** Transport réseau à haut débit (QUIC / HTTP/3).
* **Dépôt :** [`github.com/quic-go/quic-go`](https://github.com/quic-go/quic-go)
* **Description :** Implémentation autonome et complète du protocole QUIC (RFC 9000) et HTTP/3 en Pure Go, intégrée dans des proxies d'infrastructure majeurs (tels que Caddy).
* **Axe d'optimisation & Benchmarks :** Traitement à grande vitesse des datagrammes UDP, algorithmes de contrôle de congestion, parsing des en-têtes et gestion de la mémoire sous haut parallélisme.

### 7. `periph.io/x/conn`
* **Domaine :** Interface matérielle Linux embarqué.
* **Dépôt :** [`github.com/periph/conn`](https://github.com/periph/conn)
* **Description :** Bibliothèque d'abstractions et de protocoles matériels (GPIO, SPI, I2C, PWM, OneWire) en Pure Go pour cartes embarquées sous Linux.
* **Axe d'optimisation & Benchmarks :** Latence des appels système d'I/O matériel via `/dev/gpiochip` ou `/dev/i2c`, structures binaires sans allocation et manipulation directe de registres.

### 8. `google/gvisor` (`pkg/sentry`)
* **Domaine :** Virtualisation et bac à sable d'isolation (*Sandbox*).
* **Dépôt :** [`github.com/google/gvisor`](https://github.com/google/gvisor)
* **Description :** Noyau de sécurité qui émule la table des appels système Linux (projet Sentry) rédigé en Pure Go pour isoler les conteneurs sans exécuter de code C non de confiance.
* **Axe d'optimisation & Benchmarks :** Réduction du surcoût (*overhead*) des commutations d'appels système émulés, bancs d'essai de gestion de pages mémoire et verrous de concurrence.

### 9. `google/gopacket`
* **Domaine :** Analyse et forge de paquets réseau.
* **Dépôt :** [`github.com/google/gopacket`](https://github.com/google/gopacket)
* **Description :** Cadre de décodage et de traitement de couches protocolaires réseau (Ethernet, IPv4/v6, TCP, UDP, VXLAN) en Go pur.
* **Axe d'optimisation & Benchmarks :** Rapidité du décodage binaire des en-têtes de paquets, suppression totale des allocations mémoire lors de la traversée de la pile protocolaire.

### 10. `blugelabs/bluge`
* **Domaine :** Indexation et recherche textuelle haute performance.
* **Dépôt :** [`github.com/blugelabs/bluge`](https://github.com/blugelabs/bluge)
* **Description :** Moteur d'indexation et de recherche textuelle (*Full-Text Search*) écrit à 100 % en Go pur (`CGO_ENABLED=0`), conçu comme successeur moderne de Bleve.
* **Axe d'optimisation & Benchmarks :** Performance de sérialisation des segments d'index binaire sur disque, encadrement strict du *garbage collector* lors de requêtes FTS complexes.
