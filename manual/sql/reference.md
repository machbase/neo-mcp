# Machbase DBMS LLM Reference

Canonical manual: https://docs.machbase.com/dbms/index.md

## How to use the manual

Use the DBMS manual chapters as a decision tree rather than treating all SQL as generic SQL:

- Getting Started: connection checks and basic commands.
- Core Concepts: table types, time model, ROLLUP, and retention.
- Table Design: choose TAG, LOG, TRANSACTION, LOOKUP, or VOLATILE according to workload.
- TAG tables and ROLLUP: tag schema, ingestion, time-range queries, rollup design and rebuild.
- Development and Integration: client APIs and application integration.
- Security and Access Control: accounts, privileges, AUTH KEY, and access boundaries.
- Reference: exact SQL syntax, functions, configuration, and system catalogs.

## LLM query rules

- Inspect the actual table metadata before generating a query.
- Prefer Machbase-supported syntax from the Reference chapter over generic SQL assumptions.
- Use time predicates and `LIMIT` for exploratory time-series queries.
- Never infer privileges from a successful connection; the API token owner determines accessible objects and operations.

The full manual is intentionally referenced by URL here. Add focused, stable guidance to `neo-mcp/manual/` when a repeated agent failure or a Machbase-specific workflow needs a durable correction.
