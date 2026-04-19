"use client";

import { useState, useEffect } from "react";
import styles from "./page.module.css";
import { fetchAPI } from "../../lib/api";
import { PROVIDERS } from "../../lib/constants";

export default function OverviewPage() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);

  const loadData = async () => {
    try {
      const res = await fetchAPI("/admin/status");
      setData(res);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 5000);
    return () => clearInterval(interval);
  }, []);

  if (loading && !data) {
    return <div style={{ padding: "20px" }}>Loading subsystem status...</div>;
  }

  if (!data) {
    return <div style={{ padding: "20px", color: "var(--neon-red)" }}>Failed to connect to data source.</div>;
  }

  const timeline = data.timeline || { success: 0, failed: 0 };
  const keys = data.keys || [];
  const shards = data.shards || [];
  const logs = data.logs || [];

  const totalReq = timeline.success + timeline.failed;
  const successRate = totalReq > 0 ? Math.round((timeline.success / totalReq) * 100) : 0;
  const healthyKeys = keys.filter(k => k.status === "Healthy" || k.Status === "Healthy").length;

  let avgLat = 0;
  if (logs.length > 0) {
    avgLat = logs.reduce((acc, log) => acc + (log.Latency / 1000000), 0) / logs.length;
  }

  // Group by provider for brief stats
  const providerStats = {};
  keys.forEach(k => {
    const p = k.Provider || k.provider || "Unknown";
    if (!providerStats[p]) {
      providerStats[p] = { healthy: 0, total: 0, successCount: 0 };
    }
    providerStats[p].total++;
    if (k.Status === "Healthy" || k.status === "healthy" || k.Status === "active" || k.status === "active") providerStats[p].healthy++;
    providerStats[p].successCount += (k.SuccessCount || 0);
  });

  return (
    <div>
      <div className={styles.grid}>
        <div className={`glass-panel ${styles.statCard}`}>
          <div className={styles.statGlow} style={{ background: "var(--neon-blue)" }}></div>
          <div className={styles.statLabel}>Total Requests</div>
          <div className={styles.statValue} style={{ color: "var(--neon-blue)" }}>{totalReq.toLocaleString()}</div>
          <div className={styles.statSub}>{timeline.success} OK / {timeline.failed} ERR</div>
        </div>

        <div className={`glass-panel ${styles.statCard}`}>
          <div className={styles.statGlow} style={{ background: "var(--neon-green)" }}></div>
          <div className={styles.statLabel}>Success Rate</div>
          <div className={styles.statValue} style={{ color: "var(--neon-green)" }}>{successRate}%</div>
          <div className={styles.statSub}>Overall Uptime</div>
        </div>

        <div className={`glass-panel ${styles.statCard}`}>
          <div className={styles.statGlow} style={{ background: "var(--neon-purple)" }}></div>
          <div className={styles.statLabel}>Active Keys</div>
          <div className={styles.statValue} style={{ color: "var(--neon-purple)" }}>{healthyKeys}</div>
          <div className={styles.statSub}>of {keys.length} total keys</div>
        </div>

        <div className={`glass-panel ${styles.statCard}`}>
          <div className={styles.statGlow} style={{ background: "var(--neon-amber)" }}></div>
          <div className={styles.statLabel}>Avg Latency</div>
          <div className={styles.statValue} style={{ color: "var(--neon-amber)" }}>{avgLat.toFixed(0)} ms</div>
          <div className={styles.statSub}>Last 100 requests</div>
        </div>

        <div className={`glass-panel ${styles.statCard}`}>
          <div className={styles.statGlow} style={{ background: "var(--neon-red)" }}></div>
          <div className={styles.statLabel}>Database Health</div>
          <div className={styles.statValue} style={{ color: "var(--neon-red)" }}>
            {shards.filter(s => s.status === "Healthy" || s.Status === "Healthy").length} / {shards.length}
          </div>
          <div className={styles.statSub}>Shards Online</div>
        </div>
      </div>

      <div className={styles.sectionHeader}>
        Provider Health
      </div>

      <div className={styles.providerGrid}>
        {Object.keys(providerStats).map(p => {
          const meta = PROVIDERS[p] || { icon: "⚙️", color: "#666" };
          const stats = providerStats[p];
          return (
            <div key={p} className={`glass-panel ${styles.providerCard}`}>
              <div className={styles.providerIcon} style={{ color: meta.color }}>
                {meta.icon}
              </div>
              <div className={styles.providerInfo}>
                <div className={styles.providerName}>{p}</div>
                <div className={styles.providerStats}>
                  <span>{stats.healthy}/{stats.total} Healthy</span>
                  <span>{stats.successCount.toLocaleString()} Calls</span>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      <div className={styles.sectionHeader}>
        Live Activity Feed
      </div>

      <div className={styles.liveFeed}>
        <div className={styles.liveFeedWrapper}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Time</th>
                <th>Method</th>
                <th>Endpoint</th>
                <th>Status</th>
                <th>Latency</th>
                <th>Key Used</th>
              </tr>
            </thead>
            <tbody>
              {[...logs].reverse().slice(0, 15).map((log, i) => (
                <tr key={i}>
                  <td style={{ color: "var(--text-secondary)" }}>
                    {new Date(log.Timestamp).toLocaleTimeString()}
                  </td>
                  <td style={{ color: "var(--neon-blue)", fontWeight: 600 }}>{log.Method}</td>
                  <td style={{ fontFamily: "monospace", fontSize: "12px" }}>{log.Path}</td>
                  <td>
                    <span className={`${styles.statusPill} ${log.StatusCode < 400 ? styles.ok : styles.fail}`}>
                      {log.StatusCode}
                    </span>
                  </td>
                  <td style={{ color: "var(--text-secondary)" }}>{(log.Latency / 1000000).toFixed(1)}ms</td>
                  <td style={{ fontFamily: "monospace", fontSize: "11px", color: "var(--text-secondary)" }}>
                    {log.KeyUsed || "-"}
                  </td>
                </tr>
              ))}
              {logs.length === 0 && (
                <tr>
                  <td colSpan="6" style={{ textAlign: "center", padding: "24px", color: "var(--text-secondary)" }}>
                    No recent activity
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
