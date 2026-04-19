"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import styles from "./page.module.css";
import { fetchAPI } from "../../../../lib/api";

export default function DatabaseDetailsPage() {
  const params = useParams();
  const router = useRouter();
  const [dbInfo, setDbInfo] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    const fetchDetails = async () => {
      try {
        const res = await fetchAPI(`/admin/databases/${params.slug}`);
        if (res.success) {
          setDbInfo(res);
        } else {
          setError(res.message);
        }
      } catch (err) {
        setError(err.message || "Failed to load detailed analytics");
      } finally {
        setLoading(false);
      }
    };
    fetchDetails();
  }, [params.slug]);

  if (loading) return <div>Establishing direct connection...</div>;
  if (error) return <div style={{color: "var(--neon-red)"}}>{error}</div>;
  if (!dbInfo) return <div>No data found.</div>;

  const totalRows = dbInfo.tables?.reduce((acc, t) => acc + t.row_count, 0) || 0;
  const sizeMB = (dbInfo.total_bytes / (1024 * 1024)).toFixed(2);

  return (
    <div>
      <div className={styles.header}>
        <button onClick={() => router.back()} className="btn btn-glass" style={{ marginBottom: "16px" }}>
          ← Back to Topology
        </button>
        <h2>Database Analytics: {params.slug}</h2>
        <div className={styles.statusBadge}>
          Live Connection Status: <span className={`badge badge-${dbInfo.status.toLowerCase()}`}>{dbInfo.status}</span>
        </div>
      </div>

      <div className={styles.statsGrid}>
        <div className={`glass-panel ${styles.statCard}`}>
          <div className={styles.statLabel}>Ping Latency</div>
          <div className={styles.statValue} style={{ color: "var(--neon-blue)" }}>{dbInfo.latency_ms} ms</div>
        </div>
        <div className={`glass-panel ${styles.statCard}`}>
          <div className={styles.statLabel}>Total Tables</div>
          <div className={styles.statValue} style={{ color: "var(--neon-purple)" }}>{dbInfo.tables?.length || 0}</div>
        </div>
        <div className={`glass-panel ${styles.statCard}`}>
          <div className={styles.statLabel}>Total Rows</div>
          <div className={styles.statValue} style={{ color: "var(--neon-green)" }}>{totalRows.toLocaleString()}</div>
        </div>
        <div className={`glass-panel ${styles.statCard}`}>
          <div className={styles.statLabel}>Storage Usage</div>
          <div className={styles.statValue} style={{ color: "var(--neon-amber)" }}>{sizeMB} MB</div>
        </div>
      </div>

      <h3 style={{marginTop: "32px", marginBottom: "16px", fontSize: "16px"}}>Live Schema & Data Map (101% Accurate)</h3>
      <div className={styles.tableContainer}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Table Name</th>
              <th style={{textAlign: "right"}}>Exact Row Count</th>
            </tr>
          </thead>
          <tbody>
            {dbInfo.tables?.map((t) => (
              <tr key={t.name}>
                <td style={{fontFamily: "monospace", color: "var(--neon-blue)"}}>{t.name}</td>
                <td style={{textAlign: "right", fontWeight: 600}}>{t.row_count.toLocaleString()}</td>
              </tr>
            ))}
            {(!dbInfo.tables || dbInfo.tables.length === 0) && (
              <tr>
                <td colSpan="2" style={{textAlign: "center", color: "var(--text-secondary)"}}>Database is empty (no tables found).</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
