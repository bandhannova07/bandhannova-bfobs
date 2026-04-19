"use client";

import { useState } from "react";
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
        // 2. Browser WebAuthn prompt (Mocking the call to the native API for now)
        // In a production app, we use 'navigator.credentials.create'
        setMessage("Native Biometric Prompt requested. (This is a draft implementation)");
        
        // 3. Send credentials back to finish
        setTimeout(async () => {
          const final = await fetchAPI("/admin/webauthn/register/finish", { 
            method: "POST",
            body: JSON.stringify({ mock: "credential_data" }) 
          });
          if (final.success) setMessage("✅ FaceID/Fingerprint lock registered successfully!");
          setLoading(false);
        }, 1500);
      }
    } catch (err) {
      setMessage("❌ Error: " + err.message);
      setLoading(false);
    }
  };

  return (
    <div className={styles.container}>
      <div className={styles.card}>
        <div className={styles.header}>
          <div className={styles.icon}>🛡️</div>
          <div>
            <h2 className={styles.title}>Biometric Protection</h2>
            <p className={styles.subtitle}>Secure your Command Center with hardware-level authentication.</p>
          </div>
        </div>

        <div className={styles.infoBox}>
          WebAuthn (FIDO2) allows you to use your device's built-in security chip for logins. 
          Your fingerprint or facial data <strong>never leaves your device</strong>.
        </div>

        <div className={styles.actionSection}>
          <div className={styles.statusInfo}>
            <span>Status:</span>
            <span className={styles.inactive}>NOT CONFIGURED</span>
          </div>
          
          <button 
            className="btn btn-primary" 
            onClick={handleRegisterBiometric}
            disabled={loading}
          >
            {loading ? "Initializing Hardware..." : "Register FaceID / Fingerprint"}
          </button>
        </div>
      </div>

      <div className={styles.card} style={{ marginTop: "24px" }}>
        <h3 className={styles.title} style={{ fontSize: "16px" }}>Session Hardening</h3>
        <div style={{ marginTop: "16px", display: "flex", flexDirection: "column", gap: "12px" }}>
          <div className={styles.settingItem}>
            <div>
              <div className={styles.settingName}>Auto-Lock Timeout</div>
              <div className={styles.settingDesc}>Lock dashboard after 15 minutes of inactivity.</div>
            </div>
            <div className={styles.toggle}>ON</div>
          </div>
          <div className={styles.settingItem}>
            <div>
              <div className={styles.settingName}>IP Binding</div>
              <div className={styles.settingDesc}>Only allow sessions from your current IP.</div>
            </div>
            <div className={styles.toggle} style={{ color: "var(--text-secondary)" }}>OFF</div>
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
