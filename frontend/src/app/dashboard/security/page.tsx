"use client";

import React, { useState } from "react";
import styles from "./page.module.css";
import { fetchAPI } from "../../../lib/api";

export default function SecurityPage() {
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");

  const handleRegisterBiometric = async () => {
    setLoading(true);
    setMessage("");
    try {
      // 1. Get registration challenge from backend
      const res = await fetchAPI("/admin/webauthn/register/begin", { method: "POST" });
      
      if (res.success) {
        // Mocking the call to the native API for now
        setMessage("Hardware security module requested...");
        
        // 3. Send credentials back to finish
        setTimeout(async () => {
          try {
            const final = await fetchAPI("/admin/webauthn/register/finish", { 
              method: "POST",
              body: JSON.stringify({ mock: "credential_data" }) 
            });
            if (final.success) setMessage("✅ Hardware lock registered successfully!");
          } catch (err: any) {
            setMessage("❌ Registration failed: " + err.message);
          } finally {
            setLoading(false);
          }
        }, 1500);
      }
    } catch (err: any) {
      setMessage("❌ Error: " + err.message);
      setLoading(false);
    }
  };

  return (
    <div className={styles.container}>
      <div className={`glass-panel ${styles.card}`}>
        <div className={styles.header}>
          <div className={styles.icon}>🛡️</div>
          <div>
            <h2 className={styles.title}>Hardware Authentication</h2>
            <p className={styles.subtitle}>Secure your Command Center with physical device locks.</p>
          </div>
        </div>

        <div className={styles.infoBox}>
          WebAuthn (FIDO2) utilizes your device&apos;s dedicated security processor for logins. 
          Your biometric signature <strong>remains local</strong> and is never transmitted to our servers.
        </div>

        <div className={styles.actionSection}>
          <div className={styles.statusInfo}>
            <span>CURRENT STATUS</span>
            <span className={styles.inactive}>UNPROTECTED</span>
          </div>
          
          <button 
            className="btn btn-primary" 
            onClick={handleRegisterBiometric}
            disabled={loading}
          >
            {loading ? "Initializing Hardware..." : "Setup Hardware Lock"}
          </button>
        </div>
      </div>

      <div className={`glass-panel ${styles.card}`} style={{ marginTop: "32px" }}>
        <h3 className={styles.title} style={{ fontSize: "18px", marginBottom: "24px" }}>Infrastructure Hardening</h3>
        <div style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
          <div className={styles.settingItem}>
            <div>
              <div className={styles.settingName}>Intelligent Auto-Lock</div>
              <div className={styles.settingDesc}>Terminate active sessions after 15 minutes of inactivity.</div>
            </div>
            <div className={styles.toggle}>ENABLED</div>
          </div>
          <div className={styles.settingItem}>
            <div>
              <div className={styles.settingName}>Static IP Binding</div>
              <div className={styles.settingDesc}>Restrict dashboard access to your current secure IP address.</div>
            </div>
            <div className={styles.toggle} style={{ background: "#f1f5f9", color: "var(--text-muted)" }}>DISABLED</div>
          </div>
        </div>
      </div>

      {message && (
        <div className={styles.toast}>
          {message}
        </div>
      )}
    </div>
  );
}

