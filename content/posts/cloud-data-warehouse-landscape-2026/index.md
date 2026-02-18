+++
title = "The Cloud Data Warehouse Landscape in 2026"
date = "2026-02-18T00:00:00+03:00"
tags = ["data-engineering", "cloud", "benchmarking"]
categories = ["tech"]
description = """
Cloud data warehouses have never had more competition. This post breaks down the architecture, real costs, and the open format shift that changed who you should actually pick.
"""
draft = true
+++

<style>
.platform-heading {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.platform-logo {
  height: 1.4rem;
  width: auto;
  flex-shrink: 0;
  filter: brightness(0);
}
[data-theme="dark"] .platform-logo {
  filter: brightness(0) invert(1);
}
</style>

A few years ago, picking a cloud data warehouse was not much of a decision. Snowflake if you could afford it, BigQuery if you were on Google Cloud, Redshift if you were deep in AWS. The rest were either niche or still maturing.

That's no longer true. Open table formats like Apache Iceberg removed what used to be hard vendor lock-in, new entrants matured, and the "just use Snowflake" default has real challengers at every price point.

This post covers the main platforms, what they actually cost at a fixed workload, and what I'd pick for different situations.

---

## What Changed: The Open Format Shift

To understand why there's suddenly so much competition, you need to understand what the big warehouses were selling.

Traditional warehouses stored your data in their own proprietary formats. A columnar format stores data column-by-column rather than row-by-row, which makes aggregations much faster since queries only read the columns they need. Snowflake's format was good: automatic micro-partitioning (splitting data into small chunks with metadata about each, so queries can skip irrelevant chunks entirely), compression, clustering. But your data lived inside Snowflake. Migrating away was painful by design.

[Apache Iceberg](https://iceberg.apache.org/) changed that. It's an open table format that sits on top of object storage like S3 or GCS (services like Amazon S3 that store files at massive scale, cheaply). Your data stays as [Parquet](https://parquet.apache.org/) files (an open columnar file format widely used in data engineering) in your own storage bucket, with an Iceberg metadata layer on top that adds schema evolution (adding or renaming columns without breaking existing queries), time travel (querying data as it looked at a past point in time), and partition pruning (skipping irrelevant data chunks during a query).

The key shift: any query engine that supports Iceberg can read your data. Snowflake, Databricks, Spark, Trino, DuckDB. You're not locked into one vendor's runtime.

[Delta Lake](https://delta.io/), created by Databricks, solves the same problem and is the dominant format in the Databricks ecosystem.

Snowflake eventually added Iceberg table support. When the market leader adopts the open standard, you know the standard won.

<figure class="center" style="width:100%;">
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 460" width="100%" aria-label="Iceberg ecosystem diagram">
  <!-- Connection lines drawn first, behind boxes -->
  <g stroke="currentColor" stroke-opacity="0.25" stroke-width="1.5" stroke-dasharray="5 4" fill="none">
    <line x1="400" y1="240" x2="400" y2="55"/>
    <line x1="400" y1="240" x2="576" y2="183"/>
    <line x1="400" y1="240" x2="509" y2="390"/>
    <line x1="400" y1="240" x2="291" y2="390"/>
    <line x1="400" y1="240" x2="224" y2="183"/>
  </g>
  <!-- Object storage center node -->
  <rect x="310" y="208" width="180" height="64" rx="8" fill="#6C8CFF" fill-opacity="0.12" stroke="#6C8CFF" stroke-width="1.5"/>
  <text x="400" y="232" text-anchor="middle" font-family="Inter, sans-serif" font-size="12" font-weight="600" fill="#6C8CFF">Object Storage</text>
  <text x="400" y="252" text-anchor="middle" font-family="Inter, sans-serif" font-size="11" fill="#6C8CFF" opacity="0.8">Parquet · Iceberg</text>
  <!-- Snowflake -->
  <rect x="335" y="32" width="130" height="46" rx="6" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-opacity="0.2" stroke-width="1"/>
  <text x="400" y="53" text-anchor="middle" font-family="Inter, sans-serif" font-size="12" font-weight="600" fill="currentColor">Snowflake</text>
  <text x="400" y="69" text-anchor="middle" font-family="Inter, sans-serif" font-size="10" fill="currentColor" opacity="0.55">commercial warehouse</text>
  <!-- Databricks -->
  <rect x="511" y="160" width="130" height="46" rx="6" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-opacity="0.2" stroke-width="1"/>
  <text x="576" y="181" text-anchor="middle" font-family="Inter, sans-serif" font-size="12" font-weight="600" fill="currentColor">Databricks</text>
  <text x="576" y="197" text-anchor="middle" font-family="Inter, sans-serif" font-size="10" fill="currentColor" opacity="0.55">lakehouse platform</text>
  <!-- Trino -->
  <rect x="444" y="367" width="130" height="46" rx="6" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-opacity="0.2" stroke-width="1"/>
  <text x="509" y="388" text-anchor="middle" font-family="Inter, sans-serif" font-size="12" font-weight="600" fill="currentColor">Trino</text>
  <text x="509" y="404" text-anchor="middle" font-family="Inter, sans-serif" font-size="10" fill="currentColor" opacity="0.55">distributed query engine</text>
  <!-- DuckDB -->
  <rect x="226" y="367" width="130" height="46" rx="6" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-opacity="0.2" stroke-width="1"/>
  <text x="291" y="388" text-anchor="middle" font-family="Inter, sans-serif" font-size="12" font-weight="600" fill="currentColor">DuckDB</text>
  <text x="291" y="404" text-anchor="middle" font-family="Inter, sans-serif" font-size="10" fill="currentColor" opacity="0.55">in-process analytics</text>
  <!-- Apache Spark -->
  <rect x="159" y="160" width="130" height="46" rx="6" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-opacity="0.2" stroke-width="1"/>
  <text x="224" y="181" text-anchor="middle" font-family="Inter, sans-serif" font-size="12" font-weight="600" fill="currentColor">Apache Spark</text>
  <text x="224" y="197" text-anchor="middle" font-family="Inter, sans-serif" font-size="10" fill="currentColor" opacity="0.55">batch processing</text>
</svg>
<figcaption class="center">Any engine can read from the same data. Storage stays yours, compute is swappable.</figcaption>
</figure>

---

## The Commercial Platforms

<h3 class="platform-heading"><img src="snowflake.svg" class="platform-logo" alt=""> Snowflake</h3>

The benchmark everything else gets compared against.

Compute runs on virtual warehouses, clusters that spin up on demand, run your queries, and bill by the second. Storage is billed separately per TB per month. This separation of compute and storage was Snowflake's core innovation when it launched: you can scale processing power independently from how much data you hold, without paying for idle capacity overnight. Pricing is in credits, which map to warehouse size and runtime. Credits cost roughly $2-4 each depending on cloud and tier, with storage around $23/TB/month ([Snowflake pricing](https://www.snowflake.com/en/data-cloud/pricing-options/)).

Notable features worth knowing: Data Sharing, which lets you share live data with other Snowflake accounts without copying it, and Snowpark, which lets you run Python or Java directly inside Snowflake for data transformation without moving data out.

The main complaints are cost predictability (a forgotten running warehouse will run up your bill quietly) and the fact that even with Iceberg support added later, Snowflake's native format remains proprietary.

<h3 class="platform-heading"><img src="googlebigquery.svg" class="platform-logo" alt=""> BigQuery</h3>

Google's serverless data warehouse. No clusters to configure, no warehouses to size. You run a query, it runs, you get charged for how much data it scanned.

On-demand pricing at $6.25/TB scanned is intuitive at first and surprising later. A query that scans a 5TB unpartitioned table costs the same whether it returns one row or a million. The fix is partitioning your tables by a column like date, so queries only scan the slices they actually need. Ignore this and your bill will reflect it.

For predictable workloads, BigQuery offers flat-rate reservations where you buy capacity measured in slots (each slot is a unit of processing capacity) rather than paying per query. This completely changes the economics for teams running many queries daily.

Storage runs $0.02/GB/month for active data, $0.01/GB/month for data untouched for 90+ days. Compute on-demand is $6.25/TB scanned ([BigQuery pricing](https://cloud.google.com/bigquery/pricing)).

<h3 class="platform-heading"><img src="amazonwebservices.svg" class="platform-logo" alt=""> Amazon Redshift</h3>

The oldest of the main three, and it shows in parts of its design.

Redshift started as a provisioned MPP (Massively Parallel Processing) warehouse, meaning a distributed system where query work is split across many nodes simultaneously. You pick a node type and count, those nodes run around the clock, and you pay hourly whether you're using them or not. Amazon added Redshift Serverless more recently, billing by RPU-hour (Redshift Processing Unit, their measure of compute capacity) so you only pay for what you use.

Redshift Spectrum lets you query S3 files directly without loading data into Redshift first. Lake Formation and Glue integration is tight if you're already deep in AWS. It's less polished than the other two, but reserved instance discounts on provisioned clusters can make it the cheapest option when your workload is stable and predictable.

Serverless pricing is $0.375/RPU-hour, provisioned pricing varies heavily by node type ([Redshift pricing](https://aws.amazon.com/redshift/pricing/)).

<h3 class="platform-heading"><img src="databricks.svg" class="platform-logo" alt=""> Databricks</h3>

Databricks is not a data warehouse. It's a Lakehouse platform, a term they coined for the idea of combining the flexibility of a data lake (raw files in object storage) with the structure and performance of a data warehouse. It tries to be one place for data engineering, SQL analytics, machine learning, and streaming, all built on top of Delta Lake stored in your own object storage.

If your team runs Spark jobs, trains ML models, and runs SQL analytics on the same data, Databricks keeps everything in one place and the shared data model makes sense. If you just want to run SQL against structured tables, it's probably more than you need.

Compute is billed in DBUs (Databricks Units), which represent processing capacity per hour. DBU pricing varies by workload type (SQL Serverless, Jobs clusters, All-Purpose clusters) which makes cost estimation less straightforward than the other platforms. SQL Serverless runs roughly $0.22/DBU ([Databricks pricing](https://www.databricks.com/product/pricing)). Storage costs are whatever you pay for S3 or GCS, since your data lives there.

<h3 class="platform-heading"><img src="azure.svg" class="platform-logo" alt=""> Azure Synapse Analytics</h3>

Microsoft's data warehouse offering, and the right starting point if you're already invested in the Azure ecosystem.

Synapse comes in two modes. Dedicated SQL pools give you a provisioned MPP cluster that you size upfront, paying per DWU-hour (DWU standing for Data Warehouse Unit, their measure of provisioned compute). Serverless SQL pools let you query data in Azure Data Lake Storage (Azure's equivalent of S3) on demand, paying $5/TB processed, similar to BigQuery's model.

Power BI integration is tighter here than anywhere else, which matters if your organization already uses it for reporting. Synapse Link is worth knowing about: it lets you run analytics directly on operational databases like Azure Cosmos DB without running any ETL (Extract, Transform, Load) pipelines to copy the data first.

The downside is that Synapse feels like several products assembled together rather than one coherent platform. Navigating it is less intuitive than the competition. Dedicated pool pricing lands roughly $1.20-2.40/hour per 100 DWUs, Serverless at $5/TB processed ([Azure Synapse pricing](https://azure.microsoft.com/en-us/pricing/details/synapse-analytics/)).

---

## The Open Source Alternatives

<h3 class="platform-heading"><img src="duckdb.svg" class="platform-logo" alt=""> DuckDB</h3>

DuckDB is the most interesting development in analytics in recent years. It's an in-process OLAP database: OLAP (Online Analytical Processing) means designed for complex aggregations over large datasets, and in-process means no separate server, no cluster to manage, just a library you embed in your application or a binary you run locally. It can query Parquet files, CSV, JSON, and Iceberg tables directly from S3.

```sql
-- DuckDB querying S3 directly, no warehouse required
SELECT year, COUNT(*) AS trips, AVG(fare_amount) AS avg_fare
FROM read_parquet('s3://my-bucket/trips/*.parquet')
GROUP BY year
ORDER BY year;
```

For datasets that fit in memory, it's shockingly fast. For datasets that exceed memory, it has out-of-core execution that spills to disk gracefully. And it's free.

I'd reach for DuckDB first for exploratory analysis, local development, or smaller datasets. The moment you need multiple users querying a shared dataset concurrently, petabyte-scale data, or centralized access control, you'll want something else. But it's remarkable how far it gets you before that point ([DuckDB](https://duckdb.org/)).

<h3 class="platform-heading"><img src="clickhouse.svg" class="platform-logo" alt=""> ClickHouse</h3>

ClickHouse is an open-source columnar database built for a specific workload: aggregations over huge volumes of event or time-series data. User behavior analytics, metrics pipelines, log analysis. For that use case, it's the fastest option available. For general-purpose warehousing with varied query patterns, it's less of a natural fit.

Available self-hosted (free, but you manage the infrastructure) or as [ClickHouse Cloud](https://clickhouse.com/pricing) (managed). Data modeling has a learning curve compared to standard SQL warehouses, and the ecosystem is less mature than the commercial options, but the performance ceiling is hard to beat for its target workload.

<h3 class="platform-heading"><img src="trino.svg" class="platform-logo" alt=""> Trino</h3>

Trino (formerly PrestoSQL) is a distributed SQL query engine. It doesn't store data. It queries wherever your data already lives: S3, Iceberg tables, Hive Metastore (a metadata catalog used to track table schemas and file locations, originally from the Hadoop ecosystem), relational databases, Kafka. You bring the infrastructure, it brings the SQL.

If you already have data in Parquet on S3 and don't want to pay to load it into a proprietary warehouse, Trino is a compelling option. It also supports federated queries, meaning a single SQL statement that joins data from S3, a Postgres database, and a Kafka topic simultaneously, something the commercial warehouses can't do natively.

The trade-off is operational complexity. You're running a cluster yourself, or using a managed offering like [Starburst](https://www.starburst.io/). Less polished than the commercial options, more control over your data ([Trino](https://trino.io/)).

---

## Cost at a Fixed Workload

To make this concrete, I estimated monthly costs for a mid-sized workload:

- 10TB stored
- 500GB scanned per day (~15TB/month)
- ~100 queries/day, a mix of short lookups and longer aggregations

Prices sourced from each platform's public pricing pages as of early 2026. Verify before making any real decisions since these change.

| Platform | Storage/month | Compute/month | ~Total/month |
|---|---|---|---|
| Snowflake | ~$230 | ~$200-400 | **$430-630** |
| BigQuery (on-demand) | ~$200 | ~$94 (15TB × $6.25) | **~$294** |
| Redshift Serverless | ~$240 | ~$135-270 | **$375-510** |
| Databricks SQL Serverless | ~$230 (S3) | ~$200-400 | **$430-630** |
| Azure Synapse Serverless | ~$180 (ADLS) | ~$75 (15TB × $5) | **~$255** |
| ClickHouse Cloud | ~$50-100 | ~$100-200 | **$150-300** |
| DuckDB | $0 | $0 | **$0** |

A few caveats worth being upfront about. Snowflake and Databricks compute costs depend heavily on query complexity and cluster sizing, hence the wide ranges. BigQuery's compute number assumes clean partitioning: unpartitioned scans on large tables could multiply that several times. Databricks and DuckDB assume you bring your own S3. These are rough estimates, not quotes.

The broader picture: BigQuery can significantly undercut Snowflake for bursty workloads with well-partitioned tables. Azure Synapse Serverless is cheaper per TB processed than BigQuery, which surprised me given how little attention it gets. ClickHouse Cloud is competitive at mid-scale if your workload fits its strengths. DuckDB is free until it stops being the right tool.

---

## What to Pick

**Infrequent queries and no interest in managing infrastructure → BigQuery.** Pay for what you scan, nothing else. Works well when query volume is unpredictable.

**All-in on AWS → Redshift.** Spectrum for querying existing S3 data without loading it, tight Glue integration, and reserved instance savings for stable workloads.

**All-in on Azure or heavy Power BI usage → Azure Synapse.** Serverless pools for ad-hoc work, dedicated pools for production, and the tightest Power BI integration of any platform here.

**ML, data engineering, and SQL on the same data → Databricks.** The Lakehouse model makes sense when you're running the full data stack in one place.

**Open formats, full control, no lock-in → Trino or ClickHouse.** Your data stays in Iceberg on S3. You pick the query engine. More operational responsibility, less vendor dependency.

**Smaller datasets, prototyping, or embedded analytics → DuckDB.** Reach for this first. It's free and shockingly capable before you outgrow it.

**Enterprise polish, Data Sharing, mature ecosystem → Snowflake.** Still the most complete product. You pay for it.

---

## Final Thoughts

The obvious-choice era is over. Iceberg made data portable, DuckDB made local analytics viable, and the managed platforms have all closed the gap on each other's weaknesses. The decision now is less about which platform is best and more about which trade-offs fit your workload, team, and existing cloud provider.

One thing I'd think about before picking anything is where your data lives and whether you want it to stay portable. Iceberg makes that a real option now in a way it wasn't a few years ago.

Worth keeping an eye on: Iceberg adoption is accelerating and more engines are adding support. The pressure is toward commoditized compute sitting on top of open storage, which is good for buyers and bad for anyone whose business model depends on proprietary formats.

---

Thank you for reading.
