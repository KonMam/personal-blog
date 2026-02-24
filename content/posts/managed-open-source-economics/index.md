+++
title = "The Managed Open Source Bargain"
date = "2026-02-24T00:00:00+03:00"
tags = ["data-engineering", "business", "open-source"]
categories = ["tech"]
description = """
Apache Kafka is free. Airflow is free. Spark is free. But running them in production is not, and the companies that built these tools know exactly why.
"""
draft = true
+++

Using open source tools for a while, I started wondering about the commercial side. The software is free, but the companies behind it clearly aren't running on goodwill. I haven't had to make the build-vs-buy call myself, but I've worked inside enough self-hosted setups to know the complexity is real. So I dug into both sides: how these businesses actually make money, and when paying for managed genuinely makes sense.

---

## The Model

The conversion rate from free user to paying customer at open source companies is typically well below 1%. InfluxData's CEO put it plainly: ["You're a phenomenal open source company if you could monetize 1% of your community."](https://www.frontlines.io/influxdatas-playbook-how-to-convert-1-of-your-open-source-users-and-why-thats-actually-amazing/) Confluent converts a fraction of the roughly 150,000 organizations running Apache Kafka, yet built a business approaching [$1 billion in annual revenue.](https://investors.confluent.io/news-releases/news-release-details/confluent-announces-fourth-quarter-and-fiscal-year-2024) The math works because enterprise deals are large. One six-figure contract covers thousands of free users. The open source funnel replaces marketing spend; the sales team harvests the enterprise tier.

Three things usually get sold, in some combination.

<div class="diagram-scroll">
<figure class="center" style="width:100%;">
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 720 310" width="100%" aria-label="Open core product layers: open source core at the base, managed cloud service in the middle, enterprise add-ons at the top">
  <!-- Open Source Core: bottom, widest, accent -->
  <rect x="10" y="220" width="700" height="78" rx="8" fill="#6C8CFF" fill-opacity="0.12" stroke="#6C8CFF" stroke-width="1.5"/>
  <text x="360" y="247" text-anchor="middle" font-family="Inter, sans-serif" font-size="13" font-weight="600" fill="#6C8CFF">Open Source Core</text>
  <text x="360" y="266" text-anchor="middle" font-family="Inter, sans-serif" font-size="11" fill="#6C8CFF" opacity="0.85">Free · Apache 2.0 · Kafka, Airflow, Spark, dbt Core</text>
  <text x="360" y="284" text-anchor="middle" font-family="Inter, sans-serif" font-size="10" fill="#6C8CFF" opacity="0.6">Drives community adoption · less than 1% converts to paying · each paying deal is $100K–$1M+/yr</text>
  <!-- Managed Cloud Service: middle -->
  <rect x="90" y="122" width="540" height="78" rx="8" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-opacity="0.2" stroke-width="1"/>
  <text x="360" y="149" text-anchor="middle" font-family="Inter, sans-serif" font-size="13" font-weight="600" fill="currentColor">Managed Cloud Service</text>
  <text x="360" y="168" text-anchor="middle" font-family="Inter, sans-serif" font-size="11" fill="currentColor" opacity="0.6">Infrastructure, scaling, patching, SLA · billed per consumption</text>
  <text x="360" y="186" text-anchor="middle" font-family="Inter, sans-serif" font-size="10" fill="currentColor" opacity="0.4">Astro · Confluent Cloud · Databricks · dbt Cloud</text>
  <!-- Enterprise Add-ons: top, narrowest -->
  <rect x="210" y="24" width="300" height="78" rx="8" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-opacity="0.2" stroke-width="1"/>
  <text x="360" y="51" text-anchor="middle" font-family="Inter, sans-serif" font-size="13" font-weight="600" fill="currentColor">Enterprise Add-ons</text>
  <text x="360" y="70" text-anchor="middle" font-family="Inter, sans-serif" font-size="11" fill="currentColor" opacity="0.6">SSO · RBAC · audit logs · governance · compliance</text>
  <text x="360" y="88" text-anchor="middle" font-family="Inter, sans-serif" font-size="10" fill="currentColor" opacity="0.4">Annual contracts · per seat or custom deal</text>
</svg>
<figcaption class="center">The three revenue layers. The open source core is the funnel. The paid tiers are the business.</figcaption>
</figure>
</div>

Operational complexity removal is the big one. Some of these tools are genuinely painful to operate at production scale. Airflow requires choosing an executor (Celery needs Redis or RabbitMQ queues; Kubernetes needs a running cluster), managing a metadata database that degrades under heavy load, building DAG deployment pipelines by hand, and doing rolling upgrades that regularly break things. Kafka, until version 4.0 in 2024, required a separate ZooKeeper ensemble alongside every cluster: two distributed systems in production for one streaming feature. The managed service takes all of that off your plate, and for teams without dedicated platform engineers, that is worth real money.

The enterprise feature moat is less visible but just as deliberate. SSO, RBAC, audit logging, compliance certifications, paywalled almost everywhere. dbt's metadata exploration tool, cross-project reference resolution, the Semantic Layer API: all require a paid plan. Confluent's Schema Registry ships under a community license that explicitly prohibits anyone from offering it as a competing managed service. I think this is the most honest part of the model. They're not hiding what they're doing.

Compute markup is where the numbers quietly add up. Databricks bills in DBUs (Databricks Units) on top of whatever your AWS or Azure VM costs. An all-purpose cluster at the Premium tier runs about [$0.55 per DBU,](https://www.databricks.com/product/pricing) on top of EC2. Run an interactive notebook (the kind of exploratory work analysts do every day) and you pay roughly 4x the DBU rate of a scheduled batch job. The operational convenience is real. So is the difference.

---

## The Four Cases

### Airflow and Astronomer

Airflow is the dominant orchestration tool in data by a large margin: [324 million downloads in 2024](https://www.prnewswire.com/news-releases/astronomer-releases-state-of-apache-airflow-2026-report-302667480.html) alone, more than all previous years combined. Astronomer's entire business is built on the fact that most teams don't want to operate it themselves.

Astro uses usage-based pricing where workers scale to zero when idle. You only pay when tasks are actually running. That's a genuine edge over both AWS MWAA and Google Cloud Composer, which run 24/7 the moment you create an environment and can't be paused without deleting the whole thing.

| | Astronomer Astro | AWS MWAA | Cloud Composer 3 |
|---|---|---|---|
| **Pricing model** | Per execution time | Per environment-hour | Per DCU-hour |
| **Min. monthly cost** | ~$0 at idle | ~[$358](https://aws.amazon.com/managed-workflows-for-apache-airflow/pricing/) | ~[$378](https://cloud.google.com/composer/pricing) |
| **Workers scale to zero** | Yes | No | No |
| **Can be paused** | Yes | No (delete to stop) | No (delete to stop) |

Astronomer raised a [$93M Series D in May 2025](https://www.astronomer.io/press-releases/astronomer-secures-93-million-series-d-funding/) at $775M valuation, disclosing 150% year-over-year ARR growth and 130% net revenue retention. Customers don't just stay; they expand as they run more pipelines over time. Consumption-based pricing compounds in the vendor's favor when the product is embedded in daily operations.

What I find interesting about Airflow specifically is that the product's complexity isn't incidental to the business; it's structural to it. Simpler orchestration tools exist (Dagster, Prefect). Airflow persists in large organizations because of inertia and ecosystem depth. That same complexity is what makes teams reluctant to self-host, and what makes Astronomer's pitch land. It's a clean loop.

### Kafka and Confluent

Confluent was founded by the three engineers who created Kafka at LinkedIn. Jay Kreps, Confluent's CEO, has been pretty direct about the tension: ["When building a business around open source, you want to find a way to do that [that] doesn't kill all the attractiveness of an open source platform."](https://diginomica.com/confluent-continues-deliver-ceo-jay-kreps-pitches-era-data-streaming-platforms)

Apache Kafka stays fully open source under Apache 2.0. But in December 2018, when AWS announced Amazon MSK at re:Invent, Confluent moved Schema Registry and ksqlDB to a community license that prohibits anyone from running them as a competing managed service. The timing was not coincidental, and the community knew it.

Confluent Cloud is now [over 54% of total company revenue,](https://investors.confluent.io/news-releases/news-release-details/confluent-announces-third-quarter-2025-financial-results) growing 24% year-over-year as of Q3 2025. [FY2024 total revenue was $963.6M.](https://investors.confluent.io/news-releases/news-release-details/confluent-announces-fourth-quarter-and-fiscal-year-2024) For comparison, a 3-broker AWS MSK cluster on m5.large instances costs roughly [$262/month in broker compute alone,](https://aws.amazon.com/msk/pricing/) before storage, cross-AZ replication traffic, and engineering time. The MSK number looks cheap until you add all those up. That's Confluent's actual pitch and it's not wrong.

### Spark and Databricks

Databricks is approaching an IPO at a [$134B valuation,](https://www.databricks.com/company/newsroom/press-releases/databricks-grows-65-yoy-surpasses-5-4-billion-revenue-run-rate) built on top of Apache Spark. Revenue hit a $5.4B run-rate in January 2026, growing 65% year-over-year.

Ali Ghodsi, their CEO, has described the model directly: ["The traditional way of monetizing open-source technology is selling services on top of the free code, but that's not a great long-term model... Instead, have customers rent your open-source product as a service in the cloud."](https://www.battery.com/blog/billion-dollar-b2b-databricks-ali-ghodsi/) He also acknowledged the paradox: "your biggest enemy is your open-source project," because wide free adoption makes converting users to paying customers hard.

The Delta Lake strategy is the most interesting part of the Databricks story. They created it, then in 2022 fully open-sourced it to the Linux Foundation. The logic: they don't monetize the format, they monetize the compute that processes data stored in it. Keep Delta open to maximize adoption, then charge DBUs for every workload that runs against it. The proprietary moat sits in Photon (a C++ rewrite of Spark's query engine), Unity Catalog, and a full ML platform from the [$1.3B MosaicML acquisition.](https://techcrunch.com/2023/06/26/databricks-picks-up-mosaicml-an-openai-competitor-for-1-3b/) The open source parts are the funnel. The proprietary parts are the business.

### dbt and dbt Cloud

dbt Labs crossed [$100M ARR in early 2025](https://www.getdbt.com/blog/dbt-labs-100m-arr-milestone) before merging with Fivetran. dbt Cloud runs $100/seat/month on the Starter tier, with a consumption component for model runs above 15,000 per month.

dbt Core handles transformation logic and is Apache 2.0 licensed. dbt Cloud adds job scheduling, a browser-based IDE, CI/CD integration, dbt Explorer (column-level lineage, performance recommendations, documentation), and the cross-project reference resolution that makes dbt Mesh actually work. The Semantic Layer API, the integration point for BI tools to query metrics, also requires a paid plan.

The dbt case has a wrinkle the others don't. In May 2025, dbt Labs announced [dbt Fusion,](https://www.getdbt.com/blog/new-code-new-license-understanding-the-new-license-for-the-dbt-fusion-engine) a new Rust-based engine claimed to be 30x faster. It ships under Elastic License 2.0, not Apache 2.0. ELv2 is source-available but prohibits running the software as a managed service for others. dbt Core keeps its Apache license and dbt Labs committed to maintaining it, but Fusion is where the performance innovation goes. I think this is the clearest signal yet about where the model ends up: the open core gets maintained, the real development moves to something you can't self-host competitively. They just finally said it out loud. If you're adopting dbt Core today, you're betting that "actively maintained" stays true long enough to matter. At some point the performance gap gets wide enough that staying on Core becomes the lock-in, not migrating to Cloud. That point might still be a while away, but the direction is set.

---

## When Paying Makes Sense

The calculus usually comes down to engineering time. A senior data engineer costs $150-200K/year fully loaded. If managing self-hosted Airflow consumes 20% of that person's time, that's $30-40K/year in labor before incidents, upgrades, or on-call. A small Astronomer cluster might cost less. The self-hosting cost is real; it just shows up in engineering bandwidth rather than a line item on an invoice, which makes it easy to underestimate.

The case against paying gets stronger at scale. Compute markups compound. The difference between $0.30/DBU for Databricks batch jobs and $0.55/DBU for interactive compute looks small per run. Across thousands of queries daily from a team of analysts, it is not. At significant Kafka throughput, Confluent's per-unit costs can eventually exceed what a well-maintained self-managed cluster costs with one dedicated engineer.

Managed services price for the median customer. If you have the platform engineering to operate these tools well, you will eventually overpay. If you don't, you will underestimate the true cost of doing it yourself.

---

## Where It Gets Complicated

Feature migration is the quiet one. As vendors mature, capabilities that once lived in the open source core gradually appear first or only in the commercial product. This is the business model working as designed, not bad faith. But the gap between "free" and "works well at production scale" widens in one direction, consistently.

License changes are the loud version. When a hyperscaler offers your project as a managed service and captures the enterprise revenue you were building toward, the pressure to restrict the license is real. HashiCorp moved Terraform to Business Source License in 2023, prohibiting anyone from running it as a competing service. The community forked to OpenTofu within weeks, and [IBM acquired HashiCorp for $6.4B](https://medium.com/@fintanr/on-ibm-acquiring-hashicorp-c9c73a40d20c) the following year. Elastic moved to SSPL in 2021, AWS forked to create OpenSearch. Redis changed its license in 2024, the Linux Foundation launched Valkey within days backed by AWS, Google, and Oracle. Same pattern every time: backlash, a well-resourced fork, a fragmented ecosystem. The original vendor keeps the enterprise accounts. They lose the community flywheel that generates the next generation of users.

I get why they do it. Watching a hyperscaler profit from years of your open source investment, without contributing anything meaningful back, would frustrate anyone. But it does alienate the communities these companies depend on, and once that trust breaks it tends not to come back.

Pricing complexity is the one that catches teams off guard. DBUs, eCKUs, DCUs: capacity abstractions make cost comparison between vendors difficult and billing surprises easy. The engineer who leaves an interactive Databricks cluster running over a long weekend learns this lesson once.

Honestly, they're all hostile in their own way. The question isn't which vendor plays fair; none of them fully do at this stage. It's which trade-offs you can live with and which risks you're willing to carry.

---

## What to Think About Before Signing

**Pay for managed services when operational complexity is genuinely your constraint.** Teams without platform engineering depth should not be self-hosting Kafka or Airflow. The labor cost is real and usually understated.

**Model unit economics at 2x your current scale.** Get an estimate of what the steady-state bill looks like when you're twice as large, not just today. Consumption-based pricing is kind to light users and expensive to heavy ones.

**Map the feature moat before committing.** Identify which capabilities you need and which tier requires payment. The answer usually justifies the price. Occasionally it reveals the free tier covers your actual use case just fine.

**Track the license trajectory.** A vendor that has never changed its license is a different risk profile from one that has done it once. Twice is a pattern worth taking seriously before you build your stack around it.

---

Open source is one of the most effective go-to-market strategies in enterprise software: community-driven adoption, developer trust, no marketing budget needed at the top of the funnel. The managed service layer is how the economics resolve. Understanding what you're paying for, why the price is what it is, and where the vendor's incentives stop aligning with yours: that's the job before you sign anything.

Thank you for reading.
