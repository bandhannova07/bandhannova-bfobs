"use client";

import React, { useState, useEffect } from "react";
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

  // Persistence: Load from localStorage on mount
  useEffect(() => {
    const savedQuery = localStorage.getItem(`sql_forge_query_${product.slug}`);
    if (savedQuery) setQuery(savedQuery);
  }, [product.slug]);

  // Persistence: Save to localStorage on change
  const handleQueryChange = (val: string) => {
    setQuery(val);
    localStorage.setItem(`sql_forge_query_${product.slug}`, val);
  };

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

  // Extract table data from bulk results for display
  const renderShardResult = (shardSlug: string, shardResult: any) => {
    if (!shardResult) return null;

    if (!shardResult.success) {
      return (
        <div key={shardSlug} style={{ marginBottom: '16px' }}>
          <div style={{ color: '#ef4444', fontWeight: 600, marginBottom: '4px' }}>❌ {shardSlug}</div>
          <pre style={{ color: '#ef4444', fontSize: '12px', margin: 0 }}>{shardResult.error}</pre>
        </div>
      );
    }

    const res = shardResult.result;
    if (!res) return (
      <div key={shardSlug} style={{ marginBottom: '16px' }}>
        <div style={{ color: '#10b981', fontWeight: 600 }}>✅ {shardSlug}: {res?.message || "OK"}</div>
      </div>
    );

    // If there are rows, render as a table
    if (res.rows && res.rows.length > 0 && res.columns && res.columns.length > 0) {
      return (
        <div key={shardSlug} style={{ marginBottom: '20px' }}>
          <div style={{ color: '#10b981', fontWeight: 600, marginBottom: '8px' }}>
            ✅ {shardSlug} — {res.message}
          </div>
          <div style={{ overflowX: 'auto', borderRadius: '8px', border: '1px solid rgba(255,255,255,0.08)' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '12px' }}>
              <thead>
                <tr>
                  {res.columns.map((col: string) => (
                    <th key={col} style={{
                      padding: '8px 12px', textAlign: 'left',
                      background: 'rgba(0,255,136,0.05)',
                      borderBottom: '1px solid rgba(255,255,255,0.1)',
                      color: '#00ff88', fontWeight: 700, fontSize: '11px',
                      textTransform: 'uppercase', letterSpacing: '0.5px'
                    }}>{col}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {res.rows.map((row: any, idx: number) => (
                  <tr key={idx} style={{ borderBottom: '1px solid rgba(255,255,255,0.04)' }}>
                    {res.columns.map((col: string) => (
                      <td key={col} style={{
                        padding: '6px 12px',
                        color: row[col] === null ? 'rgba(255,255,255,0.25)' : 'rgba(255,255,255,0.8)',
                        fontStyle: row[col] === null ? 'italic' : 'normal'
                      }}>
                        {row[col] === null ? 'NULL' : String(row[col])}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      );
    }

    // Non-query result (INSERT/UPDATE/etc.)
    return (
      <div key={shardSlug} style={{ marginBottom: '12px' }}>
        <div style={{ color: '#10b981', fontWeight: 600 }}>✅ {shardSlug}: {res.message}</div>
      </div>
    );
  };

  return (
    <div className={styles.tabContent}>
      <div className={styles.sectionHeader}>
        <h3 className={styles.sectionTitle}>SQL Forge Console</h3>
      </div>
      <div>
        <p style={{ color: 'var(--text-secondary)', fontSize: '14px' }}>Sharded orchestration across <strong>{product.slug}</strong> infrastructure fleet.</p>
      </div>
      <div style={{ marginTop: "10px", marginBottom: "12px", display: 'flex', gap: '10px' }}>
        <button className="btn btn-glass" onClick={() => handleQueryChange("")} style={{ fontSize: '12px' }}>CLEAR</button>
        <button className="btn btn-primary" onClick={handleExecute} disabled={executing}>
          {executing ? "ORCHESTRATING..." : "EXECUTE PULSE ⚡"}
        </button>
      </div>

      <div className={`glass-panel ${styles.editorContainer}`}>
        <textarea
          className={styles.textArea}
          spellCheck="false"
          placeholder={"-- Execute across all shards in this product\nSELECT * FROM system_pulse_check;"}
          value={query}
          onChange={(e) => handleQueryChange(e.target.value)}
        ></textarea>
      </div>

      {result && (
        <div className={`glass-panel ${styles.resultArea}`}>
          <div className={styles.resultHeader}>
            <span>TERMINAL OUTPUT</span>
            <button className={styles.clearBtn} onClick={() => setResult(null)}>PURGE</button>
          </div>
          <div style={{ padding: '16px' }}>
            {result.error ? (
              <div style={{ color: '#ef4444' }}>
                <div style={{ fontWeight: 700, marginBottom: '8px' }}>❌ Execution Failed</div>
                <pre style={{ margin: 0, fontSize: '13px' }}>{result.message}</pre>
              </div>
            ) : result.results ? (
              <>
                <div style={{ color: '#10b981', marginBottom: '16px', fontWeight: 600 }}>
                  🚀 Orchestrated execution across {result.shards_executed} shard{result.shards_executed > 1 ? 's' : ''}.
                </div>
                {Object.entries(result.results).map(([slug, res]) => renderShardResult(slug, res))}
              </>
            ) : (
              <pre className={styles.pre}>{JSON.stringify(result, null, 2)}</pre>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
