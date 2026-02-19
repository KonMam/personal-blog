+++
title = "Iceberg in Practice"
date = "2026-02-19T00:00:00+02:00"
tags = ["data-engineering", "iceberg"]
categories = ["tech"]
description = """
Apache Iceberg promises portable data and freedom from vendor lock-in. This post skips the theory and shows what that actually looks like: create a table, write data, evolve the schema, travel back in time, and query it all from a second engine that had nothing to do with writing it.
"""
draft = true
+++

In my [Cloud Data Warehouse post](/posts/cloud-data-warehouse-landscape-2026/) I wrote that Iceberg removes storage lock-in: your data sits in Parquet files you own, any compatible engine can read them, you are not tied to one vendor's runtime.

I ran a local demo to test that. I create a table with [PyIceberg](https://py.iceberg.apache.org/), write some data, add a column without touching the original files, then read everything from DuckDB. No cloud account needed.

---

## How Iceberg Is Structured

Three things Iceberg manages.

<div class="diagram-scroll">
<figure class="center" style="width:100%;">
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 700 230" width="100%" aria-label="Apache Iceberg internal structure: catalog resolves table names, metadata layer tracks schema and snapshots, Parquet files hold immutable data">
  <defs>
    <marker id="iceberg-arrow" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="5" markerHeight="5" orient="auto">
      <path d="M 0 2 L 8 5 L 0 8 z" fill="currentColor" fill-opacity="0.3"/>
    </marker>
  </defs>
  <g stroke="currentColor" stroke-opacity="0.25" stroke-width="1.5" stroke-dasharray="5 4" fill="none" marker-end="url(#iceberg-arrow)">
    <!-- Catalog -> Metadata -->
    <line x1="173" y1="115" x2="212" y2="115"/>
    <!-- Metadata -> Parquet 1 -->
    <line x1="448" y1="65" x2="487" y2="55"/>
    <!-- Metadata -> Parquet 2 -->
    <line x1="448" y1="155" x2="487" y2="175"/>
  </g>
  <!-- Catalog -->
  <rect x="10" y="60" width="155" height="110" rx="6" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-opacity="0.2" stroke-width="1"/>
  <text x="87" y="82" text-anchor="middle" font-family="Inter, sans-serif" font-size="12" font-weight="600" fill="currentColor">Catalog</text>
  <text x="87" y="101" text-anchor="middle" font-family="Inter, sans-serif" font-size="10" fill="currentColor" opacity="0.55">table name registry</text>
  <text x="87" y="119" text-anchor="middle" font-family="Inter, sans-serif" font-size="9.5" fill="currentColor" opacity="0.4">shop.orders</text>
  <text x="87" y="135" text-anchor="middle" font-family="Inter, sans-serif" font-size="9.5" fill="currentColor" opacity="0.4">latest metadata path</text>
  <!-- Metadata (accent) -->
  <rect x="220" y="10" width="220" height="205" rx="8" fill="#6C8CFF" fill-opacity="0.12" stroke="#6C8CFF" stroke-width="1.5"/>
  <text x="330" y="45" text-anchor="middle" font-family="Inter, sans-serif" font-size="13" font-weight="600" fill="#6C8CFF">Metadata Layer</text>
  <text x="330" y="72" text-anchor="middle" font-family="Inter, sans-serif" font-size="10.5" fill="#6C8CFF" opacity="0.85">schema definition</text>
  <text x="330" y="92" text-anchor="middle" font-family="Inter, sans-serif" font-size="10.5" fill="#6C8CFF" opacity="0.85">snapshot history</text>
  <text x="330" y="112" text-anchor="middle" font-family="Inter, sans-serif" font-size="10.5" fill="#6C8CFF" opacity="0.85">file manifests</text>
  <text x="330" y="142" text-anchor="middle" font-family="Inter, sans-serif" font-size="9" fill="#6C8CFF" opacity="0.6">00003-....metadata.json</text>
  <!-- Parquet 1 -->
  <rect x="495" y="10" width="190" height="90" rx="6" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-opacity="0.2" stroke-width="1"/>
  <text x="590" y="34" text-anchor="middle" font-family="Inter, sans-serif" font-size="11" font-weight="600" fill="currentColor">data file 1.parquet</text>
  <text x="590" y="53" text-anchor="middle" font-family="Inter, sans-serif" font-size="10" fill="currentColor" opacity="0.55">5 rows · 4 columns</text>
  <text x="590" y="70" text-anchor="middle" font-family="Inter, sans-serif" font-size="9.5" fill="currentColor" opacity="0.4">snapshot 1</text>
  <!-- Parquet 2 -->
  <rect x="495" y="130" width="190" height="90" rx="6" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-opacity="0.2" stroke-width="1"/>
  <text x="590" y="154" text-anchor="middle" font-family="Inter, sans-serif" font-size="11" font-weight="600" fill="currentColor">data file 2.parquet</text>
  <text x="590" y="173" text-anchor="middle" font-family="Inter, sans-serif" font-size="10" fill="currentColor" opacity="0.55">3 rows · 5 columns</text>
  <text x="590" y="190" text-anchor="middle" font-family="Inter, sans-serif" font-size="9.5" fill="currentColor" opacity="0.4">snapshot 2</text>
</svg>
<figcaption class="center">The catalog resolves table names to metadata paths. The metadata layer tracks schema and snapshots. Parquet files hold the data and are never rewritten.</figcaption>
</figure>
</div>

The **catalog** is a name registry. It maps a table name like `shop.orders` to the location of that table's current metadata file.

The **metadata layer** stores the schema, snapshot history, and a manifest pointing to which Parquet files belong to each snapshot. When an engine reads a table, this is what it reads first.

The **Parquet files** are the data. Once written, never touched again. Schema changes, new writes, and deletes all produce new files or new metadata, leaving the originals alone.

---

## Setup

You need PyIceberg with PyArrow support.

```bash
pip install "pyiceberg[pyarrow,sql-sqlite]"
```

Every Iceberg table belongs to a catalog. For local work, SQLite is enough: one file on disk, no server to run.

```python
from pyiceberg.catalog.sql import SqlCatalog

catalog = SqlCatalog(
    "local",
    **{
        "uri": "sqlite:////tmp/iceberg-demo/catalog.db",
        "warehouse": "file:///tmp/iceberg-demo/warehouse",
    }
)

catalog.create_namespace("shop")
```

The `uri` is the SQLite database. The `warehouse` is where data and metadata files land on disk.

---

## Writing Data

Define a schema and create the table:

```python
from pyiceberg.schema import Schema
from pyiceberg.types import NestedField, IntegerType, FloatType, DateType

schema = Schema(
    NestedField(1, "order_id",    IntegerType()),
    NestedField(2, "customer_id", IntegerType()),
    NestedField(3, "amount",      FloatType()),
    NestedField(4, "order_date",  DateType()),
)

table = catalog.create_table("shop.orders", schema=schema)
```

One thing I hit immediately: defining fields as `required=True` expects non-nullable PyArrow arrays. Nullable arrays cause a type mismatch. Optional fields (the default) work fine.

Write the first batch:

```python
import pyarrow as pa, datetime

batch = pa.table({
    "order_id":    pa.array([1, 2, 3, 4, 5], type=pa.int32()),
    "customer_id": pa.array([101, 102, 101, 103, 102], type=pa.int32()),
    "amount":      pa.array([49.99, 120.00, 35.50, 200.00, 89.99], type=pa.float32()),
    "order_date":  pa.array([
        datetime.date(2025, 1, 5), datetime.date(2025, 1, 8),
        datetime.date(2025, 2, 1), datetime.date(2025, 2, 14),
        datetime.date(2025, 3, 3),
    ]),
})

table.append(batch)
snapshot_1_id = table.current_snapshot().snapshot_id
```

The warehouse directory now has:

```
warehouse/shop/orders/
├── data/
│   └── 457175c7-....parquet              # the actual rows
└── metadata/
    ├── 00000-....metadata.json           # table creation
    ├── 00001-....metadata.json           # state after first write
    └── snap-8806802283703698412-....avro # snapshot manifest
```

Each `append` creates a new snapshot and updates the catalog. The `iceberg_tables` row now reads:

```
catalog_name │ table_namespace │ table_name │ metadata_location
─────────────┼─────────────────┼────────────┼──────────────────────────────────
local        │ shop            │ orders     │ .../metadata/00001-....metadata.json
```

`metadata_location` is all the catalog stores: a pointer to the current metadata file. That file contains `current-snapshot-id`, which identifies the snapshot just written:

```json
{
  "current-snapshot-id": 8806802283703698412,
  "snapshots": [
    {
      "snapshot-id": 8806802283703698412,
      "manifest-list": ".../snap-8806802283703698412-....avro",
      "summary": { "added-records": "5", "total-records": "5" }
    }
  ]
}
```

When an engine reads `shop.orders`, it asks the catalog for the metadata path, opens that file, reads `current-snapshot-id`, follows the snapshot manifest, and reaches the Parquet files. The catalog pointer only moves after the new metadata file is fully written, so every write is atomic.

---

## Schema Evolution

Say the team needs to track order status. Adding a column is a metadata-only operation. The original Parquet file is never touched.

```python
with table.update_schema() as update:
    update.add_column("status", StringType())
```

Write a second batch with the new column:

```python
batch_2 = pa.table({
    "order_id":    pa.array([6, 7, 8], type=pa.int32()),
    "customer_id": pa.array([104, 101, 103], type=pa.int32()),
    "amount":      pa.array([59.00, 310.00, 45.00], type=pa.float32()),
    "order_date":  pa.array([
        datetime.date(2025, 3, 10),
        datetime.date(2025, 3, 15),
        datetime.date(2025, 4, 2),
    ]),
    "status": pa.array(["shipped", "pending", "delivered"], type=pa.string()),
})

table.append(batch_2)
```

Scan now:

```
 order_id  customer_id  amount  order_date    status
        1          101   49.99  2025-01-05      None
        2          102  120.00  2025-01-08      None
        3          101   35.50  2025-02-01      None
        4          103  200.00  2025-02-14      None
        5          102   89.99  2025-03-03      None
        6          104   59.00  2025-03-10   shipped
        7          101  310.00  2025-03-15   pending
        8          103   45.00  2025-04-02  delivered
```

Old rows return `None` for the new column. No migration job, no rewrite. I half expected this to need some kind of manual step. It does not.

---

## Time Travel

Back to snapshot 1, before `status` existed:

```python
df_old = table.scan(snapshot_id=snapshot_1_id).to_arrow()
```

```
 order_id  customer_id  amount  order_date
        1          101   49.99  2025-01-05
        2          102  120.00  2025-01-08
        3          101   35.50  2025-02-01
        4          103  200.00  2025-02-14
        5          102   89.99  2025-03-03

Columns: ['order_id', 'customer_id', 'amount', 'order_date']
```

Five rows, four columns. The table as it was before the schema change, read from the manifest for that snapshot. Useful for auditing what state a report was built on, or recovering from a bad write.

---

## Reading from DuckDB

PyIceberg wrote everything above. DuckDB just needs the metadata file path.

```sql
INSTALL iceberg; LOAD iceberg;

SELECT *
FROM iceberg_scan('/tmp/iceberg-demo/warehouse/shop/orders/metadata/00003-....metadata.json')
ORDER BY order_id;
```

```
┌──────────┬─────────────┬────────┬────────────┬───────────┐
│ order_id │ customer_id │ amount │ order_date │  status   │
├──────────┼─────────────┼────────┼────────────┼───────────┤
│        1 │         101 │  49.99 │ 2025-01-05 │ NULL      │
│        2 │         102 │ 120.00 │ 2025-01-08 │ NULL      │
│        3 │         101 │  35.50 │ 2025-02-01 │ NULL      │
│        4 │         103 │ 200.00 │ 2025-02-14 │ NULL      │
│        5 │         102 │  89.99 │ 2025-03-03 │ NULL      │
│        6 │         104 │  59.00 │ 2025-03-10 │ shipped   │
│        7 │         101 │ 310.00 │ 2025-03-15 │ pending   │
│        8 │         103 │  45.00 │ 2025-04-02 │ delivered │
└──────────┴─────────────┴────────┴────────────┴───────────┘
```

Same data. DuckDB also supports time travel:

```sql
SELECT *
FROM iceberg_scan(
  '.../metadata/00003-....metadata.json',
  snapshot_from_id=8806802283703698412
)
ORDER BY order_id;
```

```
┌──────────┬─────────────┬────────┬────────────┐
│ order_id │ customer_id │ amount │ order_date │
├──────────┼─────────────┼────────┼────────────┤
│        1 │         101 │  49.99 │ 2025-01-05 │
│        2 │         102 │ 120.00 │ 2025-01-08 │
│        3 │         101 │  35.50 │ 2025-02-01 │
│        4 │         103 │ 200.00 │ 2025-02-14 │
│        5 │         102 │  89.99 │ 2025-03-03 │
└──────────┴─────────────┴────────┴────────────┘
```

Different engine, same files, same snapshot history.

---

## The Catalog Gap

Both DuckDB queries above use a hardcoded metadata file path. That works locally but breaks in practice: the path changes with every write.

PyIceberg resolves `shop.orders` to the correct path via the SQLite catalog. DuckDB cannot read a SQLite catalog, so it takes the file path directly. In production you need a catalog that engines can actually speak to:

**[AWS Glue](https://aws.amazon.com/glue/)** is the natural fit on AWS. Athena, EMR, and Spark integrate natively.

**[Apache Polaris](https://polaris.apache.org/)** implements the Iceberg REST Catalog spec. Snowflake donated it, now open source. Most engines support it.

**[DuckLake](https://ducklake.select/)** stores all metadata in a SQL database instead of JSON files. Still early and DuckDB-centric, but worth watching.

The catalog is also where access control and audit logging live.

---

## What It Amounts To

PyIceberg wrote the files. DuckDB read them, including schema history and past snapshots, with no shared state beyond the format. The files stay put regardless of which engine touches them.

What Iceberg does not solve is everything already built around the engine you are running: dbt models, BI integrations, access policies, runbooks. Pointing a new engine at the same files does not move any of that. The switching cost is lower, not zero.

For new projects where you are deciding where data lives before building on top of it, Iceberg is worth using from the start.
