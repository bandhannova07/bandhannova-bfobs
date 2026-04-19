# [Plan] BandhanNova High-Power Database Ecosystem

Building a world-class, ultra-scalable database architecture designed for millions of users and high-concurrency API demands.

## 1. Core Architecture Strategy
We will transition from a simple multi-shard setup to a **Unified Shard Fabric** where different data types live in optimized environments.

### 1.1 Category-Based Sharding
*   **Auth Shard (Turso):** Dedicated for ecosystem-wide login/sessions. Global distribution enabled.
*   **User Shards (Turso/Horizontal):** Sharded by `fnv` hash of User ID. Scale horizontally by adding more Turso databases.
*   **Global Manager (Turso):** The "Brain" of the system. Stores system configs, API keys, and managed DB registry.
*   **Analytics Shard (Turso):** Optimized for high-write operations for logs and request tracking.

## 2. Required Additional Databases (The Next Level)

### 2.1 Caching & Speed (Redis - Upstash)
**Why:** Turso/SQLite is fast, but memory-speed is faster.
*   **Usage:** 
    *   API Rate Limiting (Distributed).
    *   User Session caching (avoiding DB hits on every request).
    *   Dynamic Config caching.
*   **Integration:** Add `internal/database/redis.go`.

### 2.2 AI Memory (Vector DB - Pinecone or Milvus)
**Why:** Standard SQL cannot "search by meaning".
*   **Usage:** 
    *   Storing AI chat embeddings for long-term user memory.
    *   Knowledge base semantic search.
*   **Integration:** Add `internal/database/vector.go`.

### 2.3 Media & Assets (Object Storage - Cloudflare R2 / S3)
**Why:** Databases should not store large binary files.
*   **Usage:** 
    *   User avatars.
    *   AI-generated images.
    *   Market data charts (static export).

## 3. Database Management Enhancements

### 3.1 The "Pulse" Monitoring System
*   Implement a background worker that pings every shard and managed DB every 60 seconds.
*   Store latency and health status in `Global Manager`.
*   Auto-failover: If a shard is down, temporarily route to a fallback or show a graceful degraded state.

### 3.2 Automated Shard Provisioning
*   Build a tool in `database_mgmt` that uses the **Turso API** to automatically create a new database shard when existing shards reach 80% capacity.

## 4. Verification Plan
*   **Latency Testing:** Verify Redis caching reduces response times from ~200ms to <20ms.
*   **Load Testing:** Simulate 10k concurrent requests across shards.
*   **Migration Verification:** Ensure cross-shard foreign key logic is handled at the Application Level.

---
> [!IMPORTANT]
> This plan moves BandhanNova from a "Project" to a "Platform". Redis and Vector DBs are essential for the AI-first features we discussed.
