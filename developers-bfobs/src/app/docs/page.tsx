"use client";
import React, { useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import styles from "./docs.module.css";

const DOCS_MARKDOWN = `# ⚡ BandhanNova Product Portal: Developer Integration Guide
> **Official Technical Documentation for Ecosystem Developers**
> *Maintain secure access to your product's infrastructure shards and cloud assets.*

---

## 1. Accessing Your Product Portal
Every product within the BandhanNova ecosystem is assigned a dedicated **Infrastructure Fleet**. You can access your management console using your **Product Credentials**:

- **Login URL:** \`/developer/login\`
- **Infrastructure ID:** Your unique product identifier (Public).
- **Security Secret:** Your private HMAC signing key (Keep this secret!).

---

## 2. Infrastructure Protocol (\`bdn-bfobs://\`)
We use a custom orchestration protocol to manage distributed shards. This allows your backend to remain agnostic of the underlying Turso database location.

**Standard Pattern (Example):**
\`bdn-bfobs://{product_slug}/{gateway_code}/gateway/\`

- **Usage:** This URL is used by the BandhanNova Global Router to resolve your active shards in real-time. 
- **Rotation:** If a database shard is migrated, the gateway code remains the same, ensuring **zero downtime**.

---

## 3. Database Proxy Gateway
Never connect directly to your Turso shards from your client-side apps. Always use the **BFOBS Proxy**.

### Execute SQL Queries
**Endpoint (Example):** \`POST /db/p/{product_slug}/execute\`

**Request Headers:**
| Header | Value | Description |
| :--- | :--- | :--- |
| \`Authorization\` | \`Bearer <Access_Token>\` | Generated from your Security Secret. |
| \`Content-Type\` | \`application/json\` | Required. |

**Request Body:**
\`\`\`json
{
  "query": "SELECT * FROM users WHERE status = ?",
  "params": ["active"]
}
\`\`\`

---

## 4. Storage & CDN (Hugging Face LFS)
Your product is automatically provisioned with an LFS-backed storage bucket.

### View/Download Files (Example)
**URL Pattern:** \`/storage/view/{product_slug}/{bucket_name}/{file_path}\`

### Uploading Assets (Example)
Use the **Storage View** in your Product Portal to manage buckets. Developers can upload assets via the portal or use the multi-tenant upload endpoint:
**Endpoint:** \`POST /storage/upload/{product_slug}/{bucket_name}\`

---

## 5. The Developer Tools (Portal Tour)
Your Product Portal contains four critical modules:

### A. Overview 📊
Your control center. View your **Gateway Credentials**, **Access Tokens**, and **Health Pulse**. Copy your integration keys directly from here.

### B. Databases 🗄️
Manage your dedicated Turso shards. You can see which shards are linked to your project and monitor their connection status.

### C. Storage ☁️
Manage your Hugging Face LFS buckets. Create folders, upload assets, and get public CDN links for your frontend.

### D. SQL Forge ⚡
A safe environment to run database migrations, test complex queries, and manage your schema across the entire fleet.

---

## 6. Integration Example (JavaScript)
\`\`\`javascript
const executeQuery = async (sql, params = []) => {
  const response = await fetch("https://api-hunter.bandhannova.in/db/p/{your-product-slug}/execute", {
    method: "POST",
    headers: {
      "Authorization": "Bearer YOUR_ACCESS_TOKEN",
      "Content-Type": "application/json"
    },
    body: JSON.stringify({ query: sql, params })
  });
  return await response.json();
};
\`\`\`

---

## 7. Security Best Practices
1. **Token Rotation:** Re-generate your **Access Token** if your Security Secret is compromised.
2. **Backend Only:** Always call the Proxy Gateway from your backend environment.
3. **Validation:** Always validate user input before passing it to the SQL Forge or Proxy Gateway.

---
**BandhanNova Infrastructure: Sharded for Performance. Secured for the Future.**`;

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
