"use client";

import { useState, useEffect } from "react";
import styles from "./page.module.css";
import { fetchAPI } from "../../../lib/api";

export default function AuditPage() {
  const [logs, setLogs] = useState([]);
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

  const getActionClass = (action) => {
    if (action.includes("ADD")) return styles["act-add"];
    if (action.includes("DELETE")) return styles["act-delete"];
    if (action.includes("CHECK")) return styles["act-check"];
    if (action.includes("LOGIN")) return styles["act-login"];
    if (action.includes("RELOAD")) return styles["act-reload"];
    return styles["act-default"];
  };

  return (
    <div>
      <div className={styles.header}>
        <div style={{ color: "var(--text-secondary)", fontSize: "14px" }}>
          Immutable Security Audit Trail
        </div>
        <button 
          className="btn btn-glass"
          onClick={loadAudit}
        >
           ↻ Refresh Log
        </button>
      </div>

      <div className={styles.tableContainer}>
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
                <td colSpan="5" style={{ textAlign: "center", padding: "40px" }}>
                  Fetching audit trail...
                </td>
              </tr>
            ) : logs.length === 0 ? (
              <tr>
                <td colSpan="5" style={{ textAlign: "center", padding: "40px" }}>
                  No audit records found.
                </td>
              </tr>
            ) : (
              logs.map((log) => (
                <tr key={log.id}>
                  <td style={{ color: "var(--text-secondary)" }}>
                    {new Date(log.timestamp * 1000).toLocaleString()}
                  </td>
                  <td>
                    <span className={`${styles.actionPill} ${getActionClass(log.action)}`}>
                      {log.action}
                    </span>
                  </td>
                  <td style={{ fontWeight: 500 }}>{log.target}</td>
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
            Showing {page * limit + 1} to {Math.min((page + 1) * limit, total)} of {total} records
          </div>
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
      )}
    </div>
  );
}
