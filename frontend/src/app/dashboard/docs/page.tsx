"use client";
import React, { useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import styles from "./docs.module.css";

const DOCS_MARKDOWN = `# ⚡ BandhanNova Infrastructure: Developer Orchestration Guide
> **Internal Technical Document - Version 1.0**
> *For use by BandhanNova Ecosystem Developers*

## 1. Overview
The **BandhanNova API Hunter (BFOBS)** is the centralized brain for managing high-performance, sharded database infrastructure across all BandhanNova products. 

Instead of a single monolithic database, every product (Blogs, Market, AI, etc.) gets its own **Dedicated Turso Shards**, all orchestrated by a global master layer. This ensures infinite scalability, edge performance, and total data isolation.

---

## 2. Core Architecture
The system operates on a **Triple-Layer Infrastructure**:

1.  **Global Manager (Global Shards):** Stores metadata about products, shard locations, and encrypted access tokens.
2.  **Core Master (Infrastructure Shards):** Manages the physical Turso databases and their health.
3.  **Product Shards (Dedicated Shards):** The actual Turso databases where product data resides.

---

## 3. Step-by-Step: Registering a New Product Backend

### Phase 1: Product Registration
1.  Navigate to **Admin Dashboard > Products**.
2.  Click **+ New Infrastructure**.
3.  Enter the **Product Name** and **Slug** (e.g., \`bandhannova-blogs\`).
4.  The system generates **OAuth 2.0 Credentials** and a **Hugging Face Storage Bucket** automatically.

### Phase 2: Linking Database Shards
1.  Go to your product's **Database View**.
2.  Click **+ Add Dedicated Shard**.
3.  Provide the **Turso Database URL** and **Access Token**.
4.  The system securely links this shard to your \`product_id\`.

---

## 4. Using the Shard Studio
- **Inspector:** Click **"Inspect"** on any shard card to open the **Shard Studio**.
- **Table Explorer:** Browse tables and schemas in the sidebar.
- **Data Grid:** View and edit rows in a spreadsheet-like interface.
- **SQL Forge:** Run bulk migrations across all shards simultaneously.

---

## 5. Developer API Integration (Proxying)
Use the **BFOBS Proxy Gateway** to execute queries securely.

**Endpoint:** \`POST /db/p/:product_slug/execute\`
**Headers:** 
- \`Authorization: Bearer <Product_OAuth_Token>\`
- \`Content-Type: application/json\`

**Body:**
\`\`\`json
{
  "query": "SELECT * FROM users WHERE email = ?",
  "params": ["dev@bandhannova.in"]
}
\`\`\`

---

## 6. Security Protocol
- **Master Key:** Required for all destructive actions.
- **Encryption:** All tokens are stored using **AES-256-GCM**.
- **Confirmation:** Requires typing **"DELETE"** for decommissioning shards or products.

---

## 5. Integration Guide: How to Use in Your Project
To integrate your project with the BandhanNova ecosystem, follow this standardized workflow:

### A. Authentication Flow
Every project must authenticate using its **Product Credentials**. 
1.  **Request Token:** Exchange your `Client ID` and `Client Secret` for a session token.
2.  **Store Token:** Keep the token in your backend environment variables (e.g., `BF_TOKEN`).

### B. Accessing Your Database
Use the **Global Proxy Gateway** instead of connecting directly to Turso. This allows BandhanNova to rotate shards without breaking your app.

**Standard URL Pattern:**
- **Proxy Gateway:** `https://api.bandhannova.in/api/db/p/:product_slug/execute`
- **Asset Storage:** `https://assets.bandhannova.in/:product_slug/:filename`

---

## 6. Security & Best Practices
Security is the backbone of the BandhanNova ecosystem. Follow these strict rules:

1.  **Never Expose Master Key:** The \`BANDHANNOVA_MASTER_KEY\` is only for administrative tasks. Never use it in client-side code or product backends.
2.  **Use Product Tokens:** Always use the OAuth token generated for your specific product.
3.  **Encrypted Transport:** All API calls MUST be made over HTTPS.
4.  **Least Privilege:** Ensure your queries only access necessary tables. Use the **Shard Studio** to test queries before deploying to production.

---

## 7. Developer API Reference

### Execute SQL (Fleet Orchestration)
Run SQL queries across your product's dedicated shard fleet.

**Endpoint:** \`POST /api/db/p/:product_slug/execute\`

| Header | Value |
| :--- | :--- |
| \`Authorization\` | \`Bearer <Your_Product_Token>\` |
| \`Content-Type\` | \`application/json\` |

**Request Body:**
\`\`\`json
{
  "query": "INSERT INTO users (name, email) VALUES (?, ?)",
  "params": ["Bandhan Developer", "dev@bandhannova.in"]
}
\`\`\`

---
**BandhanNova Infrastructure: Built for the Edge. Engineered for Scale.**`;

export default function DocsPage() {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(DOCS_MARKDOWN);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className={styles.docsContainer}>
       <div className={styles.docsHeader}>
          <div>
             <h2 className={styles.title}>System Documentation</h2>
             <p className={styles.subtitle}>Infrastructure & API Orchestration Guide</p>
          </div>
          <button className={`btn btn-primary ${styles.copyBtn}`} onClick={handleCopy}>
             {copied ? "✓ COPIED TO CLIPBOARD" : "📋 COPY FULL MARKDOWN"}
          </button>
       </div>

       <div className={`glass-panel ${styles.docsContent}`}>
          <div className={styles.markdownWrapper}>
             <ReactMarkdown remarkPlugins={[remarkGfm]}>
                {DOCS_MARKDOWN}
             </ReactMarkdown>
          </div>
       </div>
    </div>
  );
}
