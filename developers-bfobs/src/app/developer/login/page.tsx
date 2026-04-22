"use client";

import React, { useState } from "react";
import { useRouter } from "next/navigation";
import styles from "../../page.module.css"; // Reuse existing login styles
import { fetchAPI, setToken } from "../../../lib/api";

export default function DeveloperLoginPage() {
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!clientId || !clientSecret) return;

    setLoading(true);
    setError("");

    try {
      const data = await fetchAPI("/admin/developer/login", {
        method: "POST",
        body: JSON.stringify({
          client_id: clientId,
          client_secret: clientSecret
        }),
      });

      if (data.success && data.token) {
        setToken(data.token);
        // Store developer info
        sessionStorage.setItem("user_role", "developer");
        sessionStorage.setItem("product_slug", data.slug);

        // Force a full location change to ensure Layout and States re-sync correctly
        window.location.href = `/dashboard/products/${data.slug}`;
      }
    } catch (err: any) {
      setError(err.message || "Invalid Infrastructure Credentials");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className={styles.container}>
      <div className={styles.glowContainer}>
        <div className={styles.glow} style={{ background: "radial-gradient(circle, rgba(16,185,129,0.2) 0%, transparent 70%)" }}></div>
      </div>
      <div className={`glass-panel ${styles.card}`}>
        <div className={styles.logo}>
          BDN Product <b style={{ color: "#3acb9dff" }}>Portal</b>
        </div>
        <p style={{ textAlign: "center", color: "var(--text-secondary)", fontSize: "14px" }}>
          Product Infrastructure Access
        </p>

        <form className={styles.form} onSubmit={handleLogin}>
          <div className={styles.inputGroup}>
            <label>Infrastructure ID</label>
            <input
              type="text"
              className={styles.input}
              value={clientId}
              onChange={(e) => setClientId(e.target.value)}
              placeholder="bn_xxxxxxxxxxxx"
              autoFocus
            />
          </div>

          <div className={styles.inputGroup}>
            <label>Security Secret</label>
            <input
              type="password"
              className={styles.input}
              value={clientSecret}
              onChange={(e) => setClientSecret(e.target.value)}
              placeholder="Enter security secret..."
            />
          </div>

          {error && <div className={styles.error}>{error}</div>}

          <button
            type="submit"
            className={`btn btn-primary ${styles.submitBtn}`}
            style={{ background: "linear-gradient(135deg, #10b981 0%, #059669 100%)" }}
            disabled={loading}
          >
            {loading ? "Verifying..." : "ACCESS PRODUCT"}
          </button>
        </form>
      </div>
    </div>
  );
}
