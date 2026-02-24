+++
title = "The Single-Node Data Stack"
date = "2026-02-19T00:00:00+02:00"
tags = ["data-engineering", "duckdb", "polars"]
categories = ["tech"]
description = """
Most data teams reach for a managed warehouse before they need one. This post runs a full analytics pipeline on a single machine — generating 300 million rows (3 GB) with Polars and pushing DuckDB until it spills to disk.
"""
draft = true
+++

In the [Cloud Data Warehouse post](/posts/cloud-data-warehouse-landscape-2026/) I covered the managed platforms and when they make sense. This is the other side: when they do not.

Two libraries, no infrastructure, no cloud account. I generate 300 million synthetic e-commerce events with Polars, write them to Parquet, and push DuckDB through a range of query types and memory limits to find where the ceiling actually is.

---

## The Stack

[Polars](https://pola.rs/) is a DataFrame library written in Rust. It processes data lazily and in columnar batches, which makes it substantially faster than pandas on anything over a few hundred thousand rows. It is used here for data generation and transformation.

[DuckDB](https://duckdb.org/) is an in-process analytical database. It runs inside your Python process with no server, reads Parquet directly without loading everything into memory, and uses vectorized execution across all available CPU cores.

---

## Generating the Data

300 million synthetic e-commerce events: user, session, product, event type (view / add_to_cart / purchase), amount, date, device, and country. The funnel is 87% views, 9% add-to-cart, 4% purchases.

```python
import polars as pl
import numpy as np

RNG     = np.random.default_rng(42)
D_START = 19723   # 2024-01-01 as days since Unix epoch
D_END   = 20088   # 2024-12-31

def gen_batch(n: int, rng) -> pl.DataFrame:
    event_types = rng.choice(
        ["view", "add_to_cart", "purchase"], n, p=[0.87, 0.09, 0.04],
    )
    amounts = np.where(
        event_types == "purchase",
        np.round(rng.uniform(5.0, 500.0, n), 2), 0.0,
    ).astype(np.float32)

    return pl.DataFrame({
        "user_id":    rng.integers(1, 500_001,   n, dtype=np.int32),
        "session_id": rng.integers(1, 2_000_001, n, dtype=np.int32),
        "product_id": rng.integers(1, 10_001,    n, dtype=np.int32),
        "event_type": event_types,
        "amount":     amounts,
        "event_date": rng.integers(D_START, D_END + 1, n, dtype=np.int32),
        "device":  rng.choice(["mobile","desktop","tablet"], n, p=[0.55,0.38,0.07]),
        "country": rng.choice(["US","UK","DE","FR","CA","AU","JP","BR","IN","MX"], n,
                               p=[0.35,0.12,0.10,0.08,0.07,0.05,0.05,0.05,0.07,0.06]),
    }).with_columns(pl.col("event_date").cast(pl.Date))
```

The `cast(pl.Date)` converts integer days-since-epoch to a proper date column — no Python loop, entirely vectorized.

300 million rows is too large to generate in one shot. I write in 30 batches of 10M rows each, using zstd at level 3 for compression (the right default for batch analytics — covered in the [compression post](/posts/which-compression-algorithm-saves-most/)):

```python
for i in range(30):
    batch = gen_batch(10_000_000, RNG)
    batch.write_parquet(f"data/300M/batch_{i:02d}.parquet",
                        compression="zstd", compression_level=3)
```

30 batches at roughly 5.5 seconds each. 166 seconds total, **3.07 GB on disk**.

---

## DuckDB at Scale

DuckDB reads all 30 files with a glob:

```python
import duckdb
con = duckdb.connect()

con.execute("""
    SELECT month(event_date)                                 AS month,
           count(*) FILTER (WHERE event_type = 'purchase')  AS orders,
           round(sum(amount), 2)                             AS revenue
    FROM read_parquet('data/300M/*.parquet')
    GROUP BY 1 ORDER BY 1
""").df()
```

The scale benchmark runs two queries across five dataset sizes. Each result is an average of three runs.

The **simple** query is a monthly revenue aggregation. The **complex** query groups all events by user, computes per-user stats, then assigns each user to a revenue decile using a window function:

```sql
WITH user_stats AS (
    SELECT user_id,
           count(*) FILTER (WHERE event_type = 'view')     AS views,
           count(*) FILTER (WHERE event_type = 'purchase') AS purchases,
           sum(amount)                                      AS revenue
    FROM read_parquet('data/300M/*.parquet')
    GROUP BY user_id
),
ranked AS (
    SELECT *, ntile(10) OVER (ORDER BY revenue DESC) AS decile
    FROM user_stats
)
SELECT decile, count(*) AS users,
       round(avg(revenue), 2) AS avg_revenue
FROM ranked
GROUP BY 1 ORDER BY 1
```

{{< image src="chart-scale.png" alt="DuckDB query time vs dataset size: both queries scale near-linearly from 1M to 300M rows on a log-log chart" width="1500" renderWidth="900" >}}

Both queries scale close to linearly. At 300M rows — 3.07 GB of Parquet — the simple aggregation takes **1,083ms** and the complex CTE with window function takes **6,799ms**. The full table:

```
  Size   Simple   Complex
    1M      9ms     33ms
   10M     39ms    128ms
   50M    195ms    530ms
  100M    351ms  1,262ms
  300M  1,083ms  6,799ms
```

The complex query on 300M rows:

```
 decile  users  avg_revenue
      1  50000     8,694.39
      2  50000     7,554.51
      3  50000     6,989.40
      4  50000     6,560.42
      5  50000     6,181.33
      6  50000     5,822.71
      7  50000     5,460.11
      8  50000     5,065.29
      9  50000     4,585.44
     10  50000     3,714.98
```

500K unique users across 300M events, split into equal deciles by lifetime revenue. 6.8 seconds on a laptop.

---

## Pushing the Memory Limit

The interesting question is what happens when you constrain DuckDB's memory. DuckDB has a `memory_limit` setting; once exceeded it tries to spill intermediate state to disk.

Two queries tell different stories.

**CTE + window function** — builds a hash table over all 500K user groups, then applies a window function over the result. Below a certain point, the hash aggregate cannot spill and the query fails.

**Full sort** — `ORDER BY` on all 100M rows with no `LIMIT`. DuckDB must materialize and sort every row, which it handles via external merge sort when memory runs out.

{{< image src="chart-memory.png" alt="DuckDB memory pressure: CTE query on 300M rows OOMs below 512MB; ORDER BY on 100M rows degrades gracefully to 256MB" width="1800" renderWidth="1000" >}}

The CTE query runs fine down to 1GB — taking 13.2 seconds, roughly double the 6.3 seconds at no limit — then fails at 512MB. DuckDB's hash aggregate with a complex pipeline does not spill at that stage.

The sort is a different story. At 256MB — about a quarter of the 100M-row dataset's size — it finishes in 71 seconds, up from 60 seconds with no limit. A 19% slowdown. DuckDB builds sorted runs that fit in memory, writes them to a temp directory, and merges them in passes. The overhead is almost entirely disk I/O.

---

## Where This Breaks Down

The numbers show where the walls are.

**Hash aggregates don't fully spill.** The complex CTE OOMs at 512MB on 300M rows. If your workload involves high-cardinality `GROUP BY` with large intermediate state on a memory-constrained machine, you will hit OOM before you hit disk. The fix is to size memory accordingly, simplify the pipeline, or pre-aggregate upstream.

**Concurrent writes.** DuckDB allows many concurrent readers but only one writer. A solo analyst or a small team taking turns is fine. Parallel ingestion pipelines writing simultaneously is not.

**No shared access.** There is no catalog, no access control, no audit log. A file on one machine is not something a team can depend on. For that you need a [catalog layer](/posts/iceberg-in-practice/).

**Cold reads.** 3 GB of Parquet fits comfortably in memory on a modern machine, so repeated queries stay fast. Once your files exceed available RAM and stop being cached by the OS, cold query times climb. That is when the scale argument for a managed warehouse starts to make sense.

---

## What It Amounts To

300 million rows. 3.07 GB. 1.1 seconds for a full-year revenue aggregation and 6.8 seconds for a user decile analysis with a window function. $0/month.

The point is not that this replaces a warehouse. It is that most teams do not have 300 million events to analyze, and even those that do rarely query them all at once. The workload fits here.

The warehouse earns its cost when you need shared access, centralized governance, or data volumes that stop fitting on a single machine. That point is further out than most teams assume when they first reach for a managed platform.
