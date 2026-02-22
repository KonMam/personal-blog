+++
title = "The Single-Node Data Stack"
date = "2026-02-19T00:00:00+02:00"
tags = ["data-engineering", "duckdb", "polars"]
categories = ["tech"]
description = """
Most data teams reach for a managed warehouse before they need one. This post runs a full analytics pipeline on a single machine using Polars and DuckDB — generating 50M rows, benchmarking codecs, and pushing DuckDB until it spills to disk.
"""
draft = true
+++

In the [Cloud Data Warehouse post](/posts/cloud-data-warehouse-landscape-2026/) I covered the main managed platforms and when they make sense. This is the other side of that: when they do not.

Two libraries, no infrastructure, no cloud account. I generate up to 50 million e-commerce events with Polars, write them to Parquet, and push DuckDB through a range of query types and memory limits to find where the ceiling actually is.

---

## The Stack

[Polars](https://pola.rs/) is a DataFrame library written in Rust. It processes data lazily and in columnar batches, which makes it substantially faster than pandas on anything over a few hundred thousand rows. It is used here for data generation and transforms.

[DuckDB](https://duckdb.org/) is an in-process analytical database. It runs inside your Python process with no server, reads Parquet directly without loading everything into memory first, and uses vectorized execution across all available CPU cores.

---

## Codec Choice

Before generating data, it is worth picking the right Parquet compression codec. My [compression benchmarks](/posts/which-compression-algorithm-saves-most/) showed that zstd at level 3 is the right default for batch analytics: near-optimal file size with write and read speeds that match snappy.

This holds for Parquet too. Here is the comparison on 10M rows:

{{< image src="chart-codec.png" alt="Parquet codec comparison: file size and read speed by codec across snappy, lz4, zstd-1, zstd-3, zstd-9" width="1500" renderWidth="900" >}}

snappy and lz4 land at 120 MB. zstd at any level drops that to 97–98 MB — an 18% reduction — with identical read times. Write speed is slightly slower at higher levels, but zstd-9 is only 0.13s slower than snappy for 10M rows.

All datasets here use **zstd-3**.

---

## Generating the Data

Five datasets: 1M, 5M, 10M, 25M, and 50M synthetic e-commerce events. Each row has a user, session, product, event type (view / add_to_cart / purchase), amount, date, device, and country. The funnel is 87% views, 9% add-to-cart, 4% purchases.

```python
import polars as pl
import numpy as np
import datetime

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

For the larger datasets I write in batches of 5M rows to stay well within available memory:

```python
for i in range(n_batches):
    batch = gen_batch(5_000_000, RNG)
    batch.write_parquet(f"data/batch_{i:03d}.parquet",
                        compression="zstd", compression_level=3)
```

Generation times:

```
 1M    0.6s   10 MB
 5M    2.7s   49 MB
10M    5.4s   98 MB
25M   13.6s  244 MB
50M   27.3s  488 MB
```

50 million rows in 27 seconds, 488 MB on disk.

---

## DuckDB at Scale

DuckDB reads multiple Parquet files with a glob:

```python
import duckdb
con = duckdb.connect()

con.execute("""
    SELECT month(event_date)                                 AS month,
           count(*) FILTER (WHERE event_type = 'purchase')  AS orders,
           round(sum(amount), 2)                             AS revenue
    FROM read_parquet('data/50M/*.parquet')
    GROUP BY 1 ORDER BY 1
""").df()
```

The scale benchmark runs two queries across all five dataset sizes. Each result is an average of three runs after one warmup.

The **simple** query is a monthly revenue aggregation. The **complex** query groups all events by user, computes per-user stats, then assigns each user to a revenue decile using a window function:

```sql
WITH user_stats AS (
    SELECT user_id,
           count(*) FILTER (WHERE event_type = 'view')     AS views,
           count(*) FILTER (WHERE event_type = 'purchase') AS purchases,
           sum(amount)                                      AS revenue
    FROM read_parquet('data/*.parquet')
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

{{< image src="chart-scale.png" alt="DuckDB query time vs dataset size: both simple and complex queries scale near-linearly from 1M to 50M rows" width="1200" renderWidth="800" >}}

Both queries scale close to linearly. At 50M rows, the simple aggregation takes **185ms** and the complex CTE with window function takes **750ms**. The complex query on 10M rows produces:

```
 decile  users  avg_revenue  avg_views  avg_purchases
      1  50000       776.69       17.4            2.5
      2  50000       468.12       17.4            1.4
      3  50000       364.60       17.4            1.3
      4  50000       255.92       17.4            1.2
      5  50000       136.80       17.4            1.1
      6  50000        19.82       17.4            0.5
      7  50000         0.00       17.4            0.0
```

Deciles 7–10 have zero revenue — roughly half the users who ever visited never made a purchase.

---

## Pushing the Memory Limit

The interesting question is what happens when you constrain DuckDB's memory. DuckDB has a `memory_limit` setting; once exceeded it tries to spill intermediate state to disk.

Two queries tell different stories:

**CTE + window function** — builds a hash table over 500K user groups, then runs a window function over the result. The hash aggregate can handle some memory pressure but cannot spill below a certain point.

**Full sort** — `ORDER BY` on all 50M rows with no `LIMIT`. This is the worst case: DuckDB must materialize and sort every row, which it handles via external merge sort when memory runs out.

{{< image src="chart-memory.png" alt="DuckDB memory pressure on 50M rows: CTE query runs fine until OOM below 512MB; full sort completes at all limits with modest slowdown" width="1800" renderWidth="1000" >}}

The CTE query handles up to 1GB (taking 1.4 seconds, up from 780ms with no limit) and then fails with OOM. Hash aggregation in DuckDB does not fully spill — once the hash table can't fit, the query errors out.

The sort tells a different story. At 256MB — roughly 0.05% of the dataset size — it completes in 33 seconds, up from 28 seconds with no limit. DuckDB uses external merge sort: it builds sorted runs that fit in memory, writes them to a temp file on disk, and merges them in passes. The 20% slowdown reflects the disk I/O.

---

## Where This Breaks Down

The numbers show where the walls are.

**Hash aggregates don't spill.** The complex CTE fails below 512MB on 50M rows. If your workload involves high-cardinality `GROUP BY` with large intermediate state and you are on a memory-constrained machine, you will hit OOM before you hit disk. Design around it or size your memory accordingly.

**Dataset size.** 488 MB for 50M rows of this schema is small — real production data tends to be wider. At a few GB of Parquet, the OS page cache keeps things fast. Once files exceed available RAM and stop being cached, cold query times climb.

**Concurrent writers.** DuckDB allows many concurrent readers but only one writer. One analyst or a small team taking turns is fine. Parallel ingestion pipelines writing simultaneously is not.

**No shared access.** There is no catalog, no access control, no audit log here. A file on one machine is not something a team can depend on. For that you need the [catalog layer](/posts/iceberg-in-practice/).

For a solo analyst or a small team with datasets under a few GB, this stack covers most of the actual workload.

---

## What It Amounts To

50 million rows. 488 MB. Sub-second aggregations and 750ms for a full user decile analysis. $0/month.

The point is not that this replaces a warehouse — it is that most teams do not have 50 million events to analyze, and even those that do rarely query them all in a single operation. The workload fits here.

The warehouse earns its cost at the point where you need shared access, centralized governance, or data volumes that stop fitting on a single machine. That point is further out than most teams assume when they first reach for a managed platform.

Thank you for reading.
