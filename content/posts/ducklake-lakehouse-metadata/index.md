+++
title = "DuckLake: Lakehouse Metadata Without the Headaches"
date = "2026-02-19T00:00:00+02:00"
tags = ["data-engineering", "iceberg", "duckdb"]
categories = ["tech"]
description = ""
draft = true
+++

<!--
TOPICS TO COVER:
- What the catalog problem is: file-listing overhead, non-atomic operations in Iceberg at scale
- What DuckLake does differently: metadata in a SQL database (Postgres, SQLite, MotherDuck) instead of JSON/Avro files in object storage
- Why this makes catalog ops atomic and faster
- Hands-on: local setup with DuckDB + SQLite catalog
- Limitations: DuckDB/MotherDuck-centric for now, early stage (v0.3)
- How it compares to Iceberg REST catalog and Polaris
- Who it's for vs who should stick with Iceberg
-->
