# Prueba técnica — Ingesta y fan-out de eventos de inventario

> **Importante sobre el idioma.** Estas instrucciones están en español. **Todo el
> código, los identificadores, los comentarios y los mensajes de commit deben
> estar en inglés** (es una regla del equipo). La discusión durante la entrevista
> es en español; el código, en inglés.

---

## 1. Contexto

Trabajarás sobre un escenario **sintético** (no es código ni datos reales de
ningún sistema en producción). Un centro de distribución genera un alto volumen
de **eventos de movimiento de inventario**: cada vez que entra o sale stock de un
producto se emite un evento. El trabajo tiene la misma forma que buena parte de
lo que hacemos: **ingerir un alto volumen de eventos, procesarlos de forma
concurrente, persistirlos de forma correcta y exponerlos a quien los consume.**

Un evento tiene esta forma (una línea JSON por evento, formato NDJSON):

```json
{"event_id":"evt-00000004","sku":"SKU-0003","type":"OUT","quantity":20,"occurred_at":"2026-06-01T02:12:46Z"}
```

- `event_id`: identificador único del evento en el origen.
- `sku`: producto al que aplica el movimiento.
- `type`: `IN` (entrada de stock) u `OUT` (salida de stock).
- `quantity`: cantidad del movimiento (entero positivo).
- `occurred_at`: timestamp del movimiento (RFC 3339).

El **stock actual** de un producto es el resultado de aplicar todos sus
movimientos (`IN` suma, `OUT` resta).

### Qué se te entrega

```
.
├── README.md                 # este archivo
├── docker-compose.yml        # PostgreSQL listo para levantar
├── go.mod                    # módulo Go (puedes reestructurar el repo a tu gusto)
├── data/
│   ├── products.csv          # catálogo de productos (sku,name)
│   └── events/part-*.ndjson  # eventos de ejemplo
└── tools/gen/main.go         # generador de datos (para escalar el dataset)
```

Para levantar la base de datos:

```bash
docker compose up -d        # PostgreSQL en localhost:5438
```

Para ejecutar la ingesta de movimientos:

```bash
docker compose up -d
go run ./cmd/ingest
```

La ingesta usa la variable de entorno `DATABASE_URL` (opcional). Por defecto conecta a
`postgres://takehome:takehome@localhost:5438/inventory?sslmode=disable`.

Para regenerar o **escalar** el dataset (por ejemplo, para probar con volumen):

```bash
go run ./tools/gen                          # muestra pequeña (por defecto)
go run ./tools/gen -n 2000000 -files 20     # ~2M eventos repartidos en 20 archivos
```

El generador (`tools/gen`) es solo una utilidad para producir datos; **no es la
solución** y no necesitas modificarlo.

---

## 2. Qué tienes que construir

El stack del equipo es **Go** en el backend, **PostgreSQL** como base de datos y
**React + TypeScript** en el frontend. Construye sobre ese stack.

### Parte 1 — Ingesta concurrente de movimientos

Un comando que lee los archivos de `data/events/`, valida cada evento y persiste
los movimientos en PostgreSQL, dejando el stock por producto correcto.

Condiciones del entorno que debes respetar (forman parte del problema):

- En producción los archivos pueden contener **millones de eventos en total**, y
  deben procesarse **en conjunto**.
- **Procesa los archivos de forma concurrente.**
- La base de datos te da un **pool de conexiones de tamaño 10**. Tu solución debe
  funcionar correctamente respetando ese límite.
- La ingesta puede tardar mucho. **El orquestador puede cancelarla en cualquier
  momento** (timeout o `SIGINT`). Cuando eso ocurre, el proceso debe **detenerse
  de forma ordenada**, sin dejar trabajo a medias ni recursos abiertos.
- El flujo de eventos tiene **entrega _at-least-once_**: el **mismo evento (mismo
  `event_id`) puede llegar más de una vez**, dentro de un archivo o entre
  archivos. **El stock final debe ser correcto sin importar cuántas veces se
  entregue un evento.**
- **Algunas líneas vienen malformadas o con datos inválidos** (JSON roto,
  `quantity` negativa, `type` desconocido, `sku` inexistente). Deben **registrarse
  y omitirse** sin abortar todo el proceso.

### Parte 2 — Esquema y API de consulta

- **Diseña el esquema en PostgreSQL** (productos, movimientos y lo que necesites)
  y entrega las migraciones.
- Expón una **API REST** que permita:
  1. consultar el **stock actual** de cada producto;
  2. consultar el **historial de movimientos de un producto**.
- Ten en cuenta: **el historial por producto se consulta de forma constante** y la
  tabla de movimientos llegará a tener **millones de filas**. La consulta tiene
  que seguir respondiendo bien a ese volumen.

### Parte 3 — Frontend (React + TypeScript)

Una página mínima que consuma tu API de la Parte 2:

- lista de productos con su **stock actual**;
- al seleccionar un producto, su **historial de movimientos**.

> El **pulido visual no se evalúa**. Lo que importa es que consuma correctamente
> la API y que uses TypeScript de forma idiomática.

---

## 3. Tiempo y alcance

- **Tope de tiempo: 4 horas.** Es un límite firme, no un objetivo a superar.
- **Esto es a propósito más de lo que entra cómodo en 4 horas.** No esperamos que
  entregues todo. **Acota tu tiempo, prioriza, y entrega lo que tengas**, aunque
  sea parcial. **Qué decides priorizar bajo el tope es, en sí mismo, parte de lo
  que evaluamos.**
- En `DECISIONS.md` (o en el README de tu repo) cuéntanos brevemente:
  - tus **decisiones de diseño y los trade-offs** que tomaste;
  - **qué dejaste fuera por el tope de tiempo y por qué**;
  - **dónde usaste IA**: qué aceptaste, qué rechazaste y qué mejorarías.

---

## 4. Uso de IA

**El uso de IA está permitido y lo fomentamos** — buscamos acelerar el trabajo,
no prohibir herramientas. No vamos a vigilar ni a intentar detectar su uso.

Como la IA está permitida, esta prueba **no mide autoría**; mide **comprensión,
criterio y entrega**: que entendiste lo que se pedía, que el resultado es
correcto, y que detectaste dónde la salida de la IA estaba equivocada. Cuéntanos
en `DECISIONS.md` dónde la usaste y qué corregiste.

**No uses herramientas que generan mensajes de commit automáticamente a partir
del diff** (Copilot, aider u otras). Esos mensajes describen el cambio, no tu
razonamiento, y el historial pierde su valor. Los mensajes de commit deben ser
tuyos.

---

## 5. Git y entrega

Trabaja con Git **como lo harías normalmente** y haz commits de forma incremental.
No pedimos un número mínimo de commits ni un formato concreto de mensaje
(Conventional Commits, etc.). Sí esperamos:

- Mensajes que comuniquen **qué cambió y por qué importa** — como notas para un
  compañero de equipo. `wip`, `más cosas`, `updates` no aportan.
- **No hagas squash** antes de entregar y **no uses force-push.** Queremos el
  historial real.

**Cómo entregar:**

1. Crea un **repositorio en GitHub** (público o privado, a tu elección).
2. Si es **privado**, otorga acceso de lectura al usuario de GitHub del tech
   lead: **`ccromer_FTC`**.
3. **Envía el enlace del repositorio por correo a Christopher Cromer
   <ccromer@falabella.cl>** antes de la fecha límite.

**Fecha y hora límite: 25/06/2026 22:00CLT.**

---

## 6. Después de la entrega

Habrá una **revisión en vivo** breve de tu solución: nos la explicas, tomamos
alguna parte y te pedimos extenderla. **La IA está permitida también en esa
instancia.** No es un examen de autoría: es una conversación sobre las decisiones,
los trade-offs y cómo extenderías la solución.

Si tienes dudas sobre el enunciado, escribe a **ccromer@falabella.cl**. Preferimos
que preguntes a que asumas.
