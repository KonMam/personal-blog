+++
title = "kafka-replay-cli: A Lightweight Kafka Replay & Debugging Tool"
date = "2025-08-18T21:41:41+03:00"
description = "Introducing `kafka-replay-cli`, a lightweight Python tool for Kafka message replay and debugging. Learn about its features for dumping, replaying, and querying Kafka data, and the architectural decisions behind its development."
tags = ["kafka", "cli", "python"]
categories = ["tech"]
draft = false
+++

## Project Links

* **GitHub**: [github.com/KonMam/kafka-replay-cli](https://github.com/KonMam/kafka-replay-cli)
* **PyPI**: [pypi.org/project/kafka-replay-cli](https://pypi.org/project/kafka-replay-cli/)

## Why I Built This

I wanted more hands-on [Kafka](https://kafka.apache.org/) experience - that's the gist of it. Before this, I’d dealt with a few producers/consumers here and there, read the docs, and studied Kafka’s architectural design principles (very insightful read if you are interested in that sort of thing: [https://kafka.apache.org/documentation/](https://kafka.apache.org/documentation/)).
But there’s only so much you can learn with limited exposure and just reading, so I decided to spend some time tinkering and learning by doing.

## Goals

There were a few things I wanted to achieve with this:

* Get more Kafka experience - main goal.
* Integrate [DuckDB](https://duckdb.org/) - for the past year, I have seen a lot of hype around it and have started using it for some ad-hoc analysis. I enjoy using it, so I wanted to find a place for it.
* Have something to show at the end of it - meaning, find a real issue that people using Kafka might have and develop something around it, applying good practices.

## Problem & MVP

I needed to find a problem I could so-called 'solve,' even if it had been done before. After some careful Googling and ChatGPT-ing, **Kafka message replay** came up as something people either struggle with or need heavy tools to handle. The tool should be useful for someone who needs to reprocess events with filters or transformations, debugging, or migrating data between topics.

The initial MVP I scoped was simple:
* Basic replay of messages with filters.
* Ability to dump Kafka topic data.
* Query dumped data.

I wanted it lightweight, scriptable, and easy to use - no streaming engine, web UI, or over-engineering.

## Architecture

The first decision I had to make was whether to use Python or Golang.

**Arguments for Python** - I have the most experience with it and expected it would be easier and faster to develop.
**Arguments for Golang** - In the long run, it would most likely be more performant. I would get more familiar with Golang.

Due to my decision to have something tangible in a few days, I went with Python. Since it is a small tool and I didn’t know how much use it would get, I preferred not to worry about making it as performant as possible - premature optimization is the root of all evil, after all.

**Tools used for this project:**

* Kafka - the core thing I wanted to learn. Using the `confluent_kafka` Python package, as it had all the features I needed.
* DuckDB - see above.
* [Typer](https://typer.tiangolo.com/) - a library for building CLI applications. I had never used it before but liked the look and ergonomics it offered.
* [PyArrow for Parquet](https://arrow.apache.org/docs/python/parquet.html) - efficient storage; I’m used to working with it, and DuckDB can read from it. For alternatives could have used JSON or Avro, but JSON is inefficient for larger data volumes. Avro - might add support in the future.

## Features

* Dump Kafka topics into Parquet files
* Replay messages from Parquet back into Kafka
* Filter replays by timestamp range and key
* Optional throttling during replay
* Apply custom transform hooks to modify or skip messages
* Preview replays without sending messages using `--dry-run`
* Control output verbosity with `--verbose` and `--quiet`
* Query message dumps with DuckDB SQL

## Lessons Learned

**Kafka** - Not as intimidating as expected and quite enjoyable. Both the official Kafka CLI tools and the Python integrations are mature.
**DuckDB** - Currently limited use in the project, but good for what it does. I might add more use for it in the future or remove it to reduce bloat if it isn’t utilized.
**Typer** - Enjoyed working with it a lot. Super easy to get a CLI tool going.
**Testing** - Used `pytest`. For unit tests, I didn’t want Kafka running for each test, so I used `MagicMock` and `monkeypatch` to simulate real objects - techniques I’ll keep in my pocket for future. For integration testing, I spun up a Docker container with a Kafka broker to test real usage of the CLI using `subprocess`.

**Main takeaway:**
It’s important to figure out your goals and think about the architecture before you start mashing on the keyboard. Deciding the project scope and dependencies early let me focus on the main features. It’s always a balancing act: what’s core, what’s nice to have, and how much time you want to spend.

## Outcome & Reflection

Did I get more Kafka experience? Yes.
Does the tool do what I set out to make it do? Yes.
Is it the best thing since sliced bread? Highly unlikely.
Are there better tools for this use case? Probably.

At the end of the day, this was a learning experience and I had fun. If someone uses it - great. If no one does - also great, it just means that I didn't spend enough time researching real usage problems.

## Installation & Usage

```bash
pip install kafka-replay-cli
```

```bash
kafka-replay-cli dump --help
kafka-replay-cli replay --help
```

---

Thank you for reading.
