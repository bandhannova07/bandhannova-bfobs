# ⚡ BandhanNova Infrastructure: Developer Orchestration Guide
> **Internal Technical Document - Version 1.0**
> *For use by BandhanNova Ecosystem Developers*

## 1. Overview
The **BandhanNova API Hunter (BFOBS - BandhanNova Fleet Orchestration Backend System)** is the centralized brain for managing high-performance, sharded database infrastructure across all BandhanNova products. 

Instead of a single monolithic database, every product (Blogs, Market, AI, etc.) gets its own **Dedicated Turso Shards**, all orchestrated by a global master layer. This ensures infinite scalability, edge performance, and total data isolation.

---

## 2. Core Architecture
The system operates on a **Triple-Layer Infrastructure**:

1.  **Global Manager (Global Shards):** Stores metadata about products, shard locations, and encrypted access tokens.
2.  **Core Master (Infrastructure Shards):** Manages the physical Turso databases and their health.
3.  **Product Shards (Dedicated Shards):** The actual Turso databases where product data (Users, Posts, Logs) resides.

---

## 3. Step-by-Step: Registering a New Product Backend

To use this infrastructure as a backend for your product, follow these steps:

### Phase 1: Product Registration
1.  Navigate to the **Admin Dashboard > Products**.
2.  Click **+ New Infrastructure**.
3.  Enter the **Product Name** and **Slug** (e.g., `bandhannova-blogs`).
4.  The system will automatically generate **OAuth 2.0 Credentials** (Client ID & Secret) and a private **Hugging Face Storage Bucket** for your product's assets.

### Phase 2: Linking Database Shards
Since we use a manual provisioning system for maximum control:
1.  Go to your product's **Database View**.
2.  Click **+ Add Dedicated Shard**.
3.  Provide the **Turso Database URL** and **Access Token**.
4.  The system will verify the connection and securely link this shard to your `product_id`.

---

## 4. Using the Shard Studio
For schema management and data exploration, use the **BandhanNova Shard Studio**:
- **Inspector:** Click **"Inspect"** on any shard card to open the Studio.
- **Table Explorer:** Use the sidebar to browse existing tables and their schemas.
- **Data Grid:** Explore and edit rows in a professional spreadsheet-like interface.
- **SQL Forge:** Use the centralized SQL editor to run bulk migrations or complex queries across all shards of a product simultaneously.

---

## 5. Developer API Integration (Proxying)
Developers should not connect to Turso shards directly from client apps. Instead, use the **BFOBS Proxy Gateway** for security and rotation.

### Execute SQL via Proxy:
**Endpoint:** `POST /db/p/:product_slug/execute`
**Headers:** 
- `Authorization: Bearer <Your_Product_OAuth_Token>`
- `Content-Type: application/json`

**Body:**
```json
{
  "query": "SELECT * FROM users WHERE email = ?",
  "params": ["dev@bandhannova.in"]
}
```

---

## 6. Security Protocol
- **Master Key:** All destructive actions (Delete Product, Remove Shard) require the **Admin Master Key**.
- **Encryption:** All shard tokens are stored using **AES-256-GCM** encryption, keyed with the `BANDHANNOVA_MASTER_KEY`.
- **Confirmation:** Destructive actions require typing **"DELETE"** in the confirmation modal to prevent accidental data loss.

---

## 7. Troubleshooting
- **FOREIGN KEY Error:** Ensure the `product_id` exists before adding a shard.
- **Connection Failed:** Check if the Turso token is valid or has expired.
- **404 Product Not Found:** Ensure you are querying the correct Global Manager shard.

---
**BandhanNova Infrastructure: Built for the Edge. Engineered for Scale.**
