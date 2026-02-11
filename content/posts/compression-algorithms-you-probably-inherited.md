+++
title = "Compression Algorithms You Probably Inherited: gzip, Snappy, LZ4, zstd"
date = "2025-08-18T21:45:14+03:00"
description = "A deep dive into common compression algorithms like gzip, Snappy, LZ4, and zstd, explaining their trade-offs in speed, compression ratio, and ideal use cases for data engineering pipelines."
tags = ["compression", "data-engineering", "performance"]
categories = ["tech"]
draft = false
+++

## You Might Be Using The Wrong Compression Algorithm

If you work in data engineering, you’ve probably used **gzip**, **Snappy**, **LZ4**, or **Zstandard (zstd)**. More likely - you inherited them. Either the person who set these defaults is long gone, there’s never enough time to revisit the choice, or things work well enough and you’d rather not duck around and find out otherwise.

Most engineers stick with the defaults. Changing them feels risky. And let’s be honest - many don’t really know what these algorithms do or why one was chosen in the first place.

I’ve been that person myself: *"Oh, we’re using Snappy? OK."* Never thinking to ask why or what else we could use.

This post explains the most common compression algorithms, what makes them different, and when you should actually use each.

---

## Why Compression Choices Matter

Compression decisions aren’t just about saving space. They directly impact:

- **Storage costs**
- **CPU utilization**
- **Throughput**
- **Latency**

In modern pipelines — Kafka, Parquet, column stores, data lakes - the wrong compression algorithm can degrade all of these.

Two metrics matter most:
- **Compression ratio**: How much smaller the data gets.
- **Throughput**: How quickly data can be compressed and decompressed.

Your workload - and whether you prioritize CPU, latency, or bandwidth - determines which trade-offs are acceptable.

---

## Main Culprits

### gzip
- **What it is**: Uses the DEFLATE algorithm (LZ77 + Huffman coding).
- **Goal**: Good compression ratio. Compatibility.
- **Speed**: Slow to compress. Moderate decompression speed.
- **Strength**: Ubiquitous. Supported everywhere.
- **Weakness**: Outclassed in both speed and compression ratio by newer algorithms.
- **When to use**: Archival, compatibility with legacy tools. Otherwise, avoid.

### Snappy
- **What it is**: Developed by Google. Based on LZ77 without entropy coding.
- **Goal**: Maximize speed, not compression ratio.
- **Speed**: Very fast compression and decompression.
- **Strength**: Low CPU overhead. Stable. Production-proven at Google scale.
- **Weakness**: Larger compressed size than other options.
- **When to use**: Real-time, low-CPU systems where latency matters more than storage. Or if you're stuck with it.

### LZ4
- **What it is**: LZ77-based. Prioritizes speed.
- **Goal**: Fast compression and decompression with moderate compression ratio.
- **Speed**: > 500 MB/s compression. GB/s decompression.
- **Strength**: Extremely fast. Low CPU usage.
- **Weakness**: Compression ratio lower than gzip or zstd.
- **When to use**: High-throughput, low-latency systems. Datacenter transfers. OLAP engines (DuckDB, Cassandra).

### zstd (Zstandard)
- **What it is**: Developed by Facebook. Combines LZ77, Huffman coding, and FSE.
- **Goal**: High compression ratio with fast speed.
- **Speed**: Compression 500+ MB/s. Decompression ~1500+ MB/s.
- **Strength**: Tunable. Balances speed and compression. Strong performance across data types.
- **Weakness**: Slightly more CPU than LZ4/Snappy at default settings.
- **When to use**: General-purpose. Parquet files. Kafka. Data transfers. Usually the best all-around choice.

---

## Strengths and Weaknesses (At a Glance)

| Algorithm | Compression Ratio | Compression Speed | Decompression Speed | Best For |
|-----------|------------------|-------------------|---------------------|----------|
| gzip      | Moderate          | Slow              | Moderate            | Archival, web content |
| Snappy    | Low               | Very Fast         | Very Fast           | Real-time, low-CPU systems |
| LZ4       | Moderate          | Extremely Fast    | Extremely Fast      | High-throughput, low-latency systems |
| zstd      | High              | Fast              | Fast                | General-purpose, Parquet, Kafka, data transfers |

---

## Real-World Scenarios: When to Use What

### High-throughput streaming (Kafka)
- **Use**: zstd or LZ4
- **Why**: zstd gives better compression with good speed. LZ4 if latency is critical and CPU is limited. Snappy is acceptable if inherited, but usually not optimal anymore.

### Long-term storage (Parquet, S3)
- **Use**: zstd
- **Why**: Best compression ratio reduces storage cost and IO. Slight CPU trade-off is acceptable.

### Low-latency querying (DuckDB, Cassandra)
- **Use**: LZ4
- **Why**: Prioritize decompression speed for fast queries. LZ4 is the common choice in OLAP engines.

### CPU/memory constrained environments
- **Use**: Snappy or LZ4
- **Why**: Low CPU overhead is more important than compression ratio. zstd can still be used at low compression levels if needed.

### Fast network, low compression benefit (datacenter file transfer)
- **Use**: LZ4
- **Why**: Minimal compression overhead. On fast networks, speed beats smaller file sizes.

### Slow network or internet transfers
- **Use**: zstd
- **Why**: Better compression reduces transfer time despite slightly higher CPU cost.

---

## What to Remember

- No algorithm is best for every workload.
- zstd has become the Swiss Army knife of compression. Unless you have a good reason not to, it’s a smart pick.
- LZ4 is unbeatable when speed matters more than compression.
- Snappy is still acceptable in latency-sensitive, CPU-constrained setups but is generally being replaced.
- gzip remains for legacy systems or when maximum compatibility is required.

---

## What's Underneath The Hood

- **LZ77** - Replaces repeated sequences of data with references to earlier copies in the stream (sliding window). [Wikipedia](https://en.wikipedia.org/wiki/LZ77_and_LZ78)

- **Huffman Coding** - A method of assigning shorter codes to more frequent data patterns to save space. [Wikipedia](https://en.wikipedia.org/wiki/Huffman_coding)

- **FSE (Finite State Entropy)** - An advanced entropy coding method that efficiently compresses sequences by balancing speed and compression ratio. [Facebook’s zstd Manual](https://facebook.github.io/zstd/)

**Why it matters**
Most compression algorithms combine finding patterns (LZ77) with efficient encoding (Huffman, FSE) to shrink data without losing information.

---

## Closing Thoughts

Compression choices tend to stick around. There’s rarely time to revisit legacy pipelines, and if something works, it’s easy to assume it’s good enough. But if you can make the time, you’re now better equipped to review your defaults (I know I am.) - and see if a different choice might better fit your needs.

---

Thank you for reading.
