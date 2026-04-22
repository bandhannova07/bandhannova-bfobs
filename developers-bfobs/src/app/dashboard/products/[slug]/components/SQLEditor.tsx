"use client";

import React, { useState } from "react";
import styles from "../page.module.css";
import { fetchAPI } from "../../../../../lib/api";

interface Product {
  slug: string;
}

interface SQLEditorProps {
  product: Product;
}

export default function SQLEditor({ product }: SQLEditorProps) {
  const [query, setQuery] = useState("");
  const [executing, setExecuting] = useState(false);
  const [result, setResult] = useState<any>(null);

  const handleExecute = async () => {
    if (!query.trim()) return;
    setExecuting(true);
    setResult(null);
    try {
      const res = await fetchAPI("/admin/db/execute-bulk", {
        method: "POST",
        body: JSON.stringify({ product_slug: product.slug, sql: query })
      });
      setResult(res);
    } catch (err: any) {
      setResult({ error: true, message: err.message });
    } finally {
      setExecuting(false);
    }
  };

  return (
    <div className={styles.tabContent}>
      <div className={styles.editorHeader}>
        <div className={styles.headerTitle}>
          <h3>SQL Forge Console</h3>
          <p>Sharded execution across <strong>{product.slug}</strong> fleet.</p>
        </div>
        <div style={{ marginTop: "5px", marginBottom: "10px" }}>
          <button className="btn btn-primary" onClick={handleExecute} disabled={executing}>
            {executing ? "Processing..." : "Execute Pulse ⚡"}
          </button>
        </div>
      </div>

      <div className={`glass-panel ${styles.editorContainer}`}>
        <textarea
          className={styles.textArea}
          spellCheck="false"
          placeholder={"-- Select users from all infrastructure shards\nSELECT * FROM users LIMIT 10;"}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        ></textarea>
      </div>

      {result && (
        <div className={`glass-panel ${styles.resultArea}`}>
          <div className={styles.resultHeader}>
            <span>TERMINAL OUTPUT</span>
            <button className={styles.clearBtn} onClick={() => setResult(null)}>PURGE</button>
          </div>
          <pre className={styles.pre}>
            {result.shards_executed && <div style={{color:'#10b981', marginBottom:'10px'}}>🚀 Orchestrated execution successful across {result.shards_executed} shards.</div>}
            {JSON.stringify(result, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

