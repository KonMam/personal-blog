+++
title = "The Single-Node Data Stack"
date = "2026-02-19T00:00:00+02:00"
tags = ["data-engineering", "duckdb"]
categories = ["tech"]
description = ""
draft = true
+++

<!--
TOPICS TO COVER:
- ~90% of real analytical workloads don't need distributed computing
- The stack: DuckDB + Polars + Iceberg on a single machine
- Hands-on walkthrough: build a complete pipeline, ingest real data, query it
- How far it actually scales before you'd need a warehouse
- Cost comparison: $0 vs $300+/month for a managed warehouse
- Direct counterpoint to the warehouse post — when NOT to use any of those platforms
- Where the ceiling is: concurrency, dataset size, team size
-->
