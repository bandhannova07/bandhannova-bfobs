"use client";
import React, { useState, useEffect } from "react";
import styles from "../page.module.css";
import { fetchAPI } from "../../../../../lib/api";

interface Provider {
  id: string;
  name: string;
  host: string;
  port: number;
  username: string;
  from_email: string;
  status: string;
}

export default function MailView({ product }: { product: any }) {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [loading, setLoading] = useState(true);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isTestModalOpen, setIsTestModalOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [formData, setFormData] = useState({
    name: "",
    host: "",
    port: 587,
    username: "",
    password: "",
    encryption: "tls",
    from_email: ""
  });

  const [testData, setTestData] = useState({
    to: "",
    subject: "BFOBS Relay Test",
    body: "<h1>Test Successful!</h1><p>Your SMTP relay is working perfectly via BandhanNova BFOBS.</p>"
  });

  const loadProviders = async () => {
    setLoading(true);
    try {
      const res = await fetchAPI("/admin/email/providers");
      if (res.success) setProviders(res.providers || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadProviders();
  }, []);

  const handleAddProvider = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      const res = await fetchAPI("/admin/email/providers", {
        method: "POST",
        body: JSON.stringify(formData)
      });
      if (res.success) {
        setIsModalOpen(false);
        setFormData({ name: "", host: "", port: 587, username: "", password: "", encryption: "tls", from_email: "" });
        loadProviders();
      } else {
        alert(res.message);
      }
    } catch (err: any) {
      alert("Failed to add provider: " + err.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleSendTest = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      const res = await fetchAPI("/admin/email/send", {
        method: "POST",
        body: JSON.stringify({
          to: [testData.to],
          subject: testData.subject,
          html: testData.body
        })
      });
      if (res.success || !res.error) {
        alert("Test mail relayed successfully!");
        setIsTestModalOpen(false);
      } else {
        alert("Relay failed: " + res.message);
      }
    } catch (err: any) {
      alert("Relay Error: " + err.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  const deleteProvider = async (id: string) => {
    if (!confirm("Are you sure you want to remove this SMTP relay?")) return;
    try {
      const res = await fetchAPI(`/admin/email/providers/${id}`, { method: "DELETE" });
      if (res.success) loadProviders();
    } catch (err: any) {
      alert(err.message);
    }
  };

  return (
    <div className={styles.tabContent}>
      <div className={styles.sectionHeader}>
        <h3 className={styles.sectionTitle}>Mail Orchestration</h3>
        <div style={{ display: 'flex', gap: '10px' }}>
          <button className="btn btn-glass" onClick={() => setIsTestModalOpen(true)}>⚡ Test Relay</button>
          <button className="btn btn-primary" onClick={() => setIsModalOpen(true)}>+ Register SMTP Relay</button>
        </div>
      </div>

      {loading ? (
        <div className={styles.loading}>SYNCING RELAYS...</div>
      ) : providers.length === 0 ? (
        <div className={styles.emptyState}>
          <div className={styles.emptyIcon}>📧</div>
          <p>No SMTP relays configured for this product.</p>
          <button className="btn btn-primary" style={{ marginTop: '20px' }} onClick={() => setIsModalOpen(true)}>Add First Relay</button>
        </div>
      ) : (
        <div className={styles.dbGrid}>
          {providers.map(p => (
            <div key={p.id} className={`glass-panel ${styles.shardCard}`}>
              <div className={styles.shardHeader}>
                <div className={styles.shardIcon}>📬</div>
                <div className={styles.shardMeta}>
                  <h4>{p.name}</h4>
                  <span className={styles.shardStatusLabel}>{p.status.toUpperCase()} RELAY</span>
                </div>
              </div>
              <div className={styles.shardURL} style={{ fontSize: '12px', background: 'rgba(255,255,255,0.05)', padding: '8px', borderRadius: '6px' }}>
                <div>Host: {p.host}:{p.port}</div>
                <div>User: {p.username}</div>
                <div>From: {p.from_email}</div>
              </div>
              <div className={styles.shardFooter}>
                <div className={styles.shardActions}>
                  <button className="btn btn-glass" style={{ fontSize: '11px', padding: '6px 12px' }} onClick={() => alert("Edit coming soon")}>Edit</button>
                </div>
                <button className="btn btn-glass" style={{ fontSize: '11px', padding: '6px 12px', color: 'var(--danger)' }} onClick={() => deleteProvider(p.id)}>🗑️</button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* ─── Add Provider Modal ─────────────────────── */}
      {isModalOpen && (
        <div className={styles.modalOverlay}>
          <div className={`glass-panel ${styles.modalContent}`} style={{maxWidth: '500px'}}>
            <div className={styles.modalHeader}>
              <h3>Register SMTP Relay</h3>
              <p>Connect your custom mail server to the BFOBS gateway.</p>
            </div>
            <form onSubmit={handleAddProvider} className={styles.uploadForm}>
              <div className={styles.field}>
                <label>Display Name</label>
                <input className={styles.confirmInput} type="text" placeholder="e.g. BandhanNova Official"
                  value={formData.name} onChange={e => setFormData({ ...formData, name: e.target.value })} required />
              </div>
              <div style={{display: 'grid', gridTemplateColumns: '3fr 1fr', gap: '10px'}}>
                <div className={styles.field}>
                  <label>SMTP Host</label>
                  <input className={styles.confirmInput} type="text" placeholder="smtp.bandhannova.in"
                    value={formData.host} onChange={e => setFormData({ ...formData, host: e.target.value })} required />
                </div>
                <div className={styles.field}>
                  <label>Port</label>
                  <input className={styles.confirmInput} type="number" placeholder="587"
                    value={formData.port} onChange={e => setFormData({ ...formData, port: parseInt(e.target.value) })} required />
                </div>
              </div>
              <div className={styles.field}>
                <label>Username / Email</label>
                <input className={styles.confirmInput} type="text" placeholder="mail@bandhannova.in"
                  value={formData.username} onChange={e => setFormData({ ...formData, username: e.target.value })} required />
              </div>
              <div className={styles.field}>
                <label>Password</label>
                <input className={styles.confirmInput} type="password" placeholder="SMTP Password"
                  value={formData.password} onChange={e => setFormData({ ...formData, password: e.target.value })} required />
              </div>
              <div className={styles.field}>
                <label>From Email</label>
                <input className={styles.confirmInput} type="email" placeholder="mail@bandhannova.in"
                  value={formData.from_email} onChange={e => setFormData({ ...formData, from_email: e.target.value })} required />
              </div>
              <div className={styles.modalActions}>
                <button type="button" className={styles.clearBtn} onClick={() => setIsModalOpen(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
                  {isSubmitting ? "Connecting..." : "Register Relay"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ─── Test Relay Modal ───────────────────────── */}
      {isTestModalOpen && (
        <div className={styles.modalOverlay}>
          <div className={`glass-panel ${styles.modalContent}`} style={{maxWidth: '450px'}}>
            <div className={styles.modalHeader}>
              <h3>Test Mail Relay</h3>
              <p>Verify your SMTP orchestration by sending a test mail.</p>
            </div>
            <form onSubmit={handleSendTest} className={styles.uploadForm}>
              <div className={styles.field}>
                <label>Recipient Email</label>
                <input className={styles.confirmInput} type="email" placeholder="target@example.com"
                  value={testData.to} onChange={e => setTestData({ ...testData, to: e.target.value })} required />
              </div>
              <div className={styles.field}>
                <label>Subject</label>
                <input className={styles.confirmInput} type="text" value={testData.subject}
                  onChange={e => setTestData({ ...testData, subject: e.target.value })} required />
              </div>
              <div className={styles.modalActions}>
                <button type="button" className={styles.clearBtn} onClick={() => setIsTestModalOpen(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
                  {isSubmitting ? "Relaying..." : "Send Test Mail"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
