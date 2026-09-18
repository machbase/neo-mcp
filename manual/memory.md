# Agent Memory

`memory_store`, `memory_search`, and `memory_get` provide a fixed Machbase-backed memory POC.

## Workflow

Search memory before answering a task when prior decisions, incidents, procedures, or project context may be relevant. Store durable facts, important decisions, verified procedures, and useful observations. Do not store every conversational turn, credentials, API tokens, passwords, or other secrets.

Retrieved memories are lexical candidates, not authoritative facts. Check `verified`, `_arrival_time`, `source`, and provenance before relying on them.

## Storage

The tools use the internal `_NEO_AGENT_MEMORY` LOG TABLE. The table uses `TEXT` for searchable content and a KEYWORD index on `CONTENT`. `MEMORY_ID` has a scalar index for point lookup. The table is created by `memory_store` when the configured database user has the required DDL privileges.

Machbase does not provide tenant, team, author, agent, or session concepts in this POC. `tenant_id`, `team_id`, `author_id`, `agent_id`, and `session_id` are application metadata and provenance. They are not row-level security boundaries.

The database-managed `_arrival_time` is the only time reference. It is the insert time and is used for search ranges, ordering, and retention operated by the database administrator.

Machbase LOG TABLE queries do not include the `_arrival_time` pseudo-column in `*`. Select it explicitly when the query result needs the arrival time:

```sql
SELECT _arrival_time, * FROM _NEO_AGENT_MEMORY;
```

`VERIFIED` is stored in the same LOG TABLE for the POC. `CONTENT_HASH` and `SESSION_ID` are recorded for observing duplicate candidates; the POC does not perform deduplication.

## Search

`memory_search` uses `SEARCH` by default and splits the query into whitespace-separated terms. The default operator is `OR` to improve recall for LLM-generated queries. `AND` can be requested for narrower retrieval. `esearch` and `exact` are also available. `exact` uses `LIKE` and does not represent KEYWORD index retrieval.

Multi-term `SEARCH` is not phrase search. Keyword retrieval may miss records that use different words for the same meaning. Results include retrieval metadata and must not be interpreted as vector similarity scores.

## Agent use

The POC exposes memory tools to development-team Agents through MCP. The MCP server makes the backend available, but each Agent must follow the memory workflow through its instructions or tool descriptions. The backend cannot force an arbitrary Agent to store or search memories.
