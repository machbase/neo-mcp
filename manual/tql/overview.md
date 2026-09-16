# TQL LLM Reference

Canonical overview: https://docs.machbase.com/neo/tql/index.md

## Purpose

TQL is a record-flow language for reading, transforming, and emitting data. Prefer the following shape:

```tql
SOURCE(...)
MAP(...)
SINK(...)
```

The actual function catalog, signatures, argument slots, suggestions, examples, and source/map/sink roles are maintained by the neo-server TQL language metadata. Consult `neo://manual/tql` and the canonical metadata paths listed in [the base TQL manual](../tql.md).

## High-value authoring patterns

- Start with `SQL(...)`, `FAKE(...)`, `CSV(...)`, `JSON(...)`, or `SCRIPT(...)`.
- Transform records with functions such as `MAPVALUE(...)`, `FILTER(...)`, and `GROUP(...)`.
- Emit inspection results with `CSV()` or `JSON()`.
- Emit charts with `CHART(...)`; chart options use Apache ECharts option objects.
- Keep exploratory queries bounded with `LIMIT`.
- Use `value(index)`, `key(index)`, and `param(name)` instead of guessing record or request access APIs.

## Examples worth loading into context

The upstream overview is useful for these patterns:

- generating synthetic signal data with `FAKE(oscillator(...))`;
- reading SQL, CSV, JSON, or SCRIPT sources interchangeably;
- mapping records and emitting CSV/JSON;
- creating ECharts output with `CHART(chartOption({...}))`.

Use the upstream page for the full examples rather than copying the entire documentation into every prompt.
