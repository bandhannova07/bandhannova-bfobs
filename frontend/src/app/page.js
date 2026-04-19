"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import styles from "./page.module.css";
import { fetchAPI, setToken } from "../lib/api";

export default function LoginPage() {
  const [masterKey, setMasterKey] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const handleLogin = async (e) => {
    e.preventDefault();
    if (!masterKey) return;

    setLoading(true);
    setError("");

    try {
      const data = await fetchAPI("/admin/login", {
        method: "POST",
        body: JSON.stringify({ master_key: masterKey }),
      });

      if (data.success && data.token) {
        setToken(data.token);
        // Add artificial slight delay for smooth transition
        setTimeout(() => {
          router.push("/dashboard");
        }, 300);
      }
    } catch (err) {
      setError(err.message || "Authentication failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className={styles.container}>
      <div className={styles.glowContainer}>
        <div className={styles.glow}></div>
      </div>
      <div className={`glass-panel ${styles.card}`}>
        <div className={styles.logo}>
          BandhanNova <span>BFOBS</span>
        </div>
        <p style={{ textAlign: "center", color: "var(--text-secondary)", fontSize: "14px" }}>
          Command Center Authentication
        </p>
        
        <form className={styles.form} onSubmit={handleLogin}>
          <div className={styles.inputGroup}>
            <label>Master Key</label>
            <input
              type="password"
              className={styles.input}
              value={masterKey}
              onChange={(e) => setMasterKey(e.target.value)}
              placeholder="Enter ecosystem key..."
              autoFocus
            />
          </div>
          
          {error && <div className={styles.error}>{error}</div>}
          
          <button 
            type="submit" 
            className={`btn btn-primary ${styles.submitBtn}`}
            disabled={loading}
          >
            {loading ? "Authenticating..." : "ATTEMPT ACCESS"}
          </button>
        </form>
      </div>
    </div>
  );
}
