"use client";

import React, { useState, useEffect } from "react";
import styles from "./page.module.css";
import { fetchAPI } from "../../../lib/api";

interface AuditLog {
  id: string;
  timestamp: number;
  action: string;
  target: string;
  ip_address: string;
  details: string;
}

export default function AuditPage() {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const limit = 50;

  const loadAudit = async () => {
    setLoading(true);
    try {
      const offset = page * limit;
      const res = await fetchAPI(`/admin/audit?limit=${limit}&offset=${offset}`);
      if (res.success) {
        setLogs(res.logs);
        setTotal(res.total);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadAudit();
  }, [page]);

  const getActionClass = (action: string) => {
    if (action.includes("ADD")) return styles["act-add"];
    if (action.includes("DELETE")) return styles["act-delete"];
    if (action.includes("CHECK")) return styles["act-check"];
    if (action.includes("LOGIN")) return styles["act-login"];
    if (action.includes("RELOAD")) return styles["act-reload"];
    return styles["act-default"];
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div>
          <h1 style={{ fontSize: 24, fontWeight: 800, color: "var(--text-primary)" }}>Security Audit</h1>
          <p style={{ color: "var(--text-muted)", fontSize: 13, fontWeight: 600 }}>Immutable infrastructure event stream.</p>
        </div>
        <button 
          className="btn btn-glass"
          onClick={loadAudit}
        >
           ↻ Refresh Log
        </button>
      </div>

      <div className={`glass-panel ${styles.tableContainer}`}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Timestamp</th>
              <th>Action</th>
              <th>Target</th>
              <th>IP Address</th>
              <th>Details</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={5} style={{ textAlign: "center", padding: "100px", color: "var(--primary)", fontWeight: 700 }}>
                  SYNCING AUDIT STREAM...
                </td>
              </tr>
            ) : logs.length === 0 ? (
              <tr>
                <td colSpan={5} style={{ textAlign: "center", padding: "100px", color: "var(--text-muted)" }}>
                  No security events found.
                </td>
              </tr>
            ) : (
              logs.map((log) => (
                <tr key={log.id}>
                  <td style={{ color: "var(--text-muted)", fontSize: 12 }}>
                    {new Date(log.timestamp * 1000).toLocaleString()}
                  </td>
                  <td>
                    <span className={`${styles.actionPill} ${getActionClass(log.action)}`}>
                      {log.action}
                    </span>
                  </td>
                  <td style={{ fontWeight: 700 }}>{log.target}</td>
                  <td className={styles.ipCell}>{log.ip_address}</td>
                  <td className={styles.detailsText} title={log.details}>
                    {log.details}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {total > 0 && (
        <div className={styles.pagination}>
          <div className={styles.pageInfo}>
            Displaying {page * limit + 1}—{Math.min((page + 1) * limit, total)} of {total} records
          </div>
          <div className={styles.pageActions}>
            <button 
              className="btn btn-glass" 
              disabled={page === 0 || loading}
              onClick={() => setPage(page - 1)}
            >
              Previous
            </button>
            <button 
              className="btn btn-glass" 
              disabled={(page + 1) * limit >= total || loading}
              onClick={() => setPage(page + 1)}
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

