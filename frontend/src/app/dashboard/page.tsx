"use client";

import { useState, useEffect } from "react";
import styles from "./page.module.css";
import { fetchAPI } from "../../lib/api";

interface DashboardStats {
  total_requests: number;
  success_count: number;
  failed_count: number;
  active_keys: number;
  total_keys: number;
  avg_latency_ms: number;
}

interface Provider {
  name: string;
  icon?: string;
  keys: number;
  requests: number;
  success: number;
}

interface LogEntry {
  timestamp: number;
  method: string;
  card_name?: string;
  status: number;
  latency: number;
}

interface Shard {
  name: string;
  type: string;
  status: string;
}

interface DashboardData {
  stats: DashboardStats;
  providers: Provider[];
  recent_logs: LogEntry[];
  shards: Shard[];
}

export default function OverviewPage() {
  const [data, setData] = useState<DashboardData | null>(null);
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
    const interval = setInterval(loadData, 8000);
    return () => clearInterval(interval);
  }, []);

  if (loading && !data) {
    return (
      <div className={styles.loadingState}>
        <div className={styles.loadingPulse}></div>
        <span>Initializing Command Center...</span>
      </div>
    );
  }

  if (!data || !data.stats) {
    return <div className={styles.errorState}>⚠ Failed to connect to data source. Please check backend connectivity.</div>;
  }

  const s = data.stats;
  const logs = data.recent_logs || [];
  const providers = data.providers || [];
  const shards = data.shards || [];
  const successRate = s.total_requests > 0 ? Math.round((s.success_count / s.total_requests) * 100) : 0;

  return (
    <div className={styles.page}>
      {/* ─── Top Stat Cards ─────────────────────────── */}
      <div className={styles.grid}>
        <StatCard
          label="Total Requests"
          value={s.total_requests.toLocaleString()}
          sub={`${s.success_count} OK / ${s.failed_count} ERR`}
          color="var(--primary)"
          icon="📡"
        />
        <StatCard
          label="Success Rate"
          value={`${successRate}%`}
          sub="Overall Uptime"
          color="var(--success)"
          icon="✓"
        />
        <StatCard
          label="Active Keys"
          value={s.active_keys.toString()}
          sub={`of ${s.total_keys} total keys`}
          color="#8b5cf6"
          icon="🔑"
        />
        <StatCard
          label="Avg Latency"
          value={`${Math.round(s.avg_latency_ms)} ms`}
          sub="Last 100 requests"
          color="var(--warning)"
          icon="⚡"
        />
      </div>

      {/* ─── Provider Health ──────────────────────────── */}
      <div className={styles.sectionHeader}>
        <span>⚙ Provider Health</span>
        <span className={styles.sectionBadge}>{providers.length} active</span>
      </div>

      <div className={styles.providerGrid}>
        {providers.map((p, i) => (
          <div key={i} className={`glass-panel ${styles.providerCard}`}>
            <div className={styles.providerHeader}>
              <div className={styles.providerIcon}>{p.icon || "🔌"}</div>
              <div className={styles.providerInfo}>
                <div className={styles.providerName}>{p.name}</div>
                <div className={styles.providerStats}>
                  <span>{p.keys} key{p.keys !== 1 ? "s" : ""}</span>
                  <span className={styles.dot}>·</span>
                  <span>{p.requests.toLocaleString()} req</span>
                  <span className={styles.dot}>·</span>
                  <span style={{ color: "var(--success)" }}>
                    {p.requests > 0 ? Math.round((p.success / p.requests) * 100) : 0}% ok
                  </span>
                </div>
              </div>
            </div>
            <div className={styles.providerBar}>
              <div
                className={styles.providerBarFill}
                style={{ width: `${p.requests > 0 ? Math.round((p.success / p.requests) * 100) : 0}%` }}
              />
            </div>
          </div>
        ))}
        {providers.length === 0 && (
          <div className={`glass-panel ${styles.emptyCard}`}>
            No provider activity yet. Add API keys to get started.
          </div>
        )}
      </div>

      {/* ─── Live Activity Feed ──────────────────────── */}
      <div className={styles.sectionHeader}>
        <span>📡 Live Activity Feed</span>
        <span className={styles.sectionBadge}>{logs.length} recent</span>
      </div>

      <div className={styles.liveFeed}>
        <div className={styles.liveFeedWrapper}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Time</th>
                <th>Method</th>
                <th>Provider</th>
                <th>Status</th>
                <th>Latency</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((log, i) => (
                <tr key={i}>
                  <td className={styles.cellMuted}>
                    {new Date(log.timestamp * 1000).toLocaleTimeString()}
                  </td>
                  <td className={styles.cellMethod}>{log.method}</td>
                  <td className={styles.cellProvider}>{log.card_name || "—"}</td>
                  <td>
                    <span className={`${styles.statusPill} ${log.status < 400 ? styles.ok : styles.fail}`}>
                      {log.status}
                    </span>
                  </td>
                  <td className={styles.cellMuted}>{log.latency}ms</td>
                </tr>
              ))}
              {logs.length === 0 && (
                <tr>
                  <td colSpan={5} className={styles.emptyRow}>
                    No recent activity — waiting for first request
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* ─── Shard Health ─────────────────────────────── */}
      <div className={styles.sectionHeader} style={{ marginTop: 40 }}>
        <span>🗄️ Shard Health</span>
      </div>
      <div className={styles.shardGrid}>
        {shards.map((sh, i) => (
          <div key={i} className={`glass-panel ${styles.shardCard}`}>
            <div className={styles.shardDot} data-status={sh.status === "Healthy" ? "ok" : "err"} />
            <div>
              <div className={styles.shardName}>{sh.name}</div>
              <div className={styles.shardType}>{sh.type}</div>
            </div>
            <div className={styles.shardStatus}>
              <span className={`badge ${sh.status === "Healthy" ? "badge-healthy" : sh.status === "Disconnected" ? "badge-dead" : "badge-warning"}`}>
                {sh.status}
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

interface StatCardProps {
  label: string;
  value: string;
  sub: string;
  color: string;
  icon: string;
}

function StatCard({ label, value, sub, color, icon }: StatCardProps) {
  return (
    <div className={`glass-panel ${styles.statCard}`}>
      <div className={styles.statGlow} style={{ background: color }}></div>
      <div className={styles.statIcon} style={{ color }}>{icon}</div>
      <div className={styles.statLabel}>{label}</div>
      <div className={styles.statValue}>{value}</div>
      <div className={styles.statSub}>{sub}</div>
    </div>
  );
}


