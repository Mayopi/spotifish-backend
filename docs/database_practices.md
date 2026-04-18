# Database Practices

Spotifish leverages a strictly typed, explicitly defined data-access strategy relying solely on standard PostgreSQL conventions, bypassing heavy ORMs to prioritize predictable queries, maximum throughput, and low abstraction cost.

## Connection Pooling with `pgxpool`

Spotifish avoids standard `database/sql` mapping due to limitations in PostgreSQL-specific networking, electing instead to utilize raw `github.com/jackc/pgx/v5`.
For performance, connections are not explicitly managed; the system utilizes `pgxpool`, which inherently mitigates TCP connection latency by establishing multiplexed socket reserves out-of-the-box. This ensures minimal overhead for continuous CRUD execution operations and mitigates thread exhaustion under stress.

The pool singleton is spawned directly at `main.go` and injected downstream precisely into the Repository layer abstractions.

## Raw SQL vs Object-Relational Mappers

Spotifish completely rejects traditional ORM tooling (like GORM) for several essential reasons:
1.  **Complexity:** ORMs create heavy abstraction barriers for complex aggregates causing "N+1" lookup pitfalls. Spotifish manages complex entity aggregation (like HomeSections grouping) exclusively via explicit SQL `JOIN` or native subqueries.
2.  **Native Tools:** Utilizing driver-exclusive operations like standard Postgres Full-Text operations `ts_vector`/`ts_query` on search fields becomes native without hacky ORM dialect overwriting.
3.  **Explicit Tuning:** Memory allocation rules within scanning steps are heavily curated (`scanSongs` helper map loops).

## Conflict Handling (Upsert)

In typical distributed content synchronization (Drive ID checks), Spotifish avoids executing multiple data access hops (e.g. `SELECT` -> if missing -> `INSERT` -> else -> `UPDATE`).
Instead, database integrity locks resolve constraints natively using SQL `ON CONFLICT` declarations against strictly validated Unique Tables.

```sql
INSERT INTO songs (...) VALUES (...)
ON CONFLICT (user_id, source, source_file_id) DO UPDATE SET
    title = EXCLUDED.title,
    artist = EXCLUDED.artist ...
RETURNING id, added_at
```

## Cursor-based Pagination Strategies

To preserve strict query consistency over continually mutating entity subsets (like dynamically modifying user libraries where traditional `LIMIT/OFFSET` paging would inevitably skip chunks), Pagination explicitly handles ties through compound keys or secondary Tie-breaker rows (Cursor).

In standard tables:
1.  Clients explicitly provide `nextCursor` tokens denoting the exact last-perceived entity slice boundary.
2.  The persistence layer executes relative comparative scans (`WHERE id > $2`) on the specific subset rather than executing massive full table bypasses avoiding expensive row computation and indexing bypasses.
3.  An exact size + 1 (e.g., requesting limit `100` pulls `101`) is checked. The auxiliary boundary object validates the presence of an existent following subset.

## Migration Pipeline

Rather than relying on active synchronization mappings to mutate server tables locally upon deploy, Spotifish handles schema revisions as explicit, ordered, idempotent DDL scripts stored under the root `migrations/` directory.
During process bootstrap, `github.com/golang-migrate/migrate` acts natively over the connection pool ensuring any database instances conform completely to internal representation versions *before* connection locks open up traffic to HTTP routers.
