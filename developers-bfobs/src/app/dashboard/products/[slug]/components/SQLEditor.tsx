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
          <div style={{display:'flex', alignItems:'center', gap:'12px', marginBottom:'8px'}}>
             <div style={{fontSize:'24px', background:'rgba(0,255,136,0.1)', padding:'10px', borderRadius:'12px', border:'1px solid rgba(0,255,136,0.2)'}}>⚡</div>
             <h3 style={{fontSize:'22px', fontWeight:800}}>SQL Forge Console</h3>
          </div>
          <p style={{color:'var(--text-secondary)', fontSize:'14px'}}>Sharded orchestration across <strong>{product.slug}</strong> infrastructure fleet.</p>
        </div>
        <div style={{ marginTop: "10px" }}>
          <button className="btn btn-primary" onClick={handleExecute} disabled={executing} style={{background:'linear-gradient(135deg, #00ff88 0%, #00a3ff 100%)', border:'none', padding:'12px 24px'}}>
            {executing ? "ORCHESTRATING..." : "EXECUTE PULSE ⚡"}
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

