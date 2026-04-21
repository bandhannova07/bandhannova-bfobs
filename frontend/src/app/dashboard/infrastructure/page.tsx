"use client";

import React, { useState, useEffect } from "react";
import styles from "./page.module.css";
import { getShards, addShard, removeShard } from "../../../lib/api";

interface Shard {
  id: string;
  name: string;
  type: string;
  db_url: string;
  status: string;
  created_at: number;
}

const getTypeIcon = (type: string) => {
  switch (type) {
    case "global_manager": return "🌐";
    case "auth": return "🔐";
    case "analytics": return "📈";
    case "user": return "👥";
    default: return "💾";
  }
};

const getTypeColor = (type: string) => {
  switch (type) {
    case "global_manager": return "#00c6ff";
    case "auth": return "#f39c12";
    case "analytics": return "#9b59b6";
    case "user": return "#2ecc71";
    default: return "#fff";
  }
};

export default function InfrastructurePage() {
  const [shards, setShards] = useState<Shard[]>([]);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [formData, setFormData] = useState({
    name: "",
    type: "global_manager",
    db_url: "",
    token: ""
  });

  useEffect(() => {
    loadShards();
  }, []);

  const loadShards = async () => {
    setIsLoading(true);
    try {
      const data = await getShards();
      if (data.success) {
        setShards(data.shards || []);
      }
    } catch (error) {
      console.error("Failed to load shards", error);
    }
    setIsLoading(false);
  };

  const handleAddShard = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      const res = await addShard(formData);
      if (res.success) {
        setIsModalOpen(false);
        setFormData({ name: "", type: "global_manager", db_url: "", token: "" });
        loadShards();
      }
    } catch (error: any) {
      alert(error.message || "Failed to register shard. Verify credentials.");
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDeleteShard = async (id: string) => {
    if (!confirm("Are you sure you want to decommission this shard? It will be disconnected from the fleet brain.")) return;
    try {
      const res = await removeShard(id);
      if (res.success) {
        loadShards();
      }
    } catch (error) {
      alert("Failed to remove shard");
    }
  };

  return (
    <div className={styles.container}>
      <div className={styles.heroSection}>
        <div className={styles.heroContent}>
          <h2 className={styles.title}>Infrastructure Fleet</h2>
          <p className={styles.subtitle}>Manage the master shards orchestrating the BandhanNova ecosystem.</p>
        </div>
        <button onClick={() => setIsModalOpen(true)} className={styles.addBtn}>
          <span className={styles.plusIcon}>+</span> Register Master Shard
        </button>
      </div>

      {isLoading ? (
        <div className={styles.loadingContainer}>
          <div className={styles.spinner}></div>
          <p>Scanning Infrastructure Fleet...</p>
        </div>
      ) : (
        <div className={styles.grid}>
          {/* Core Master Card (Virtual representation of the env-based one) */}
          <div className={`${styles.card} ${styles.coreCard}`}>
            <div className={styles.cardGlow}></div>
            <div className={styles.cardHeader}>
              <div className={styles.typeIcon}>🧠</div>
              <div className={styles.headerInfo}>
                <span className={styles.shardName}>Core Master</span>
                <span className={styles.shardTypeLabel}>SYSTEM BRAIN</span>
              </div>
            </div>
            <div className={styles.shardURL}>[MANAGED VIA HF SECRETS]</div>
            <div className={styles.cardFooter}>
              <div className={styles.status}>
                <div className={`${styles.statusDot} ${styles.pulse}`}></div>
                Operational
              </div>
              <div className={styles.readOnlyTag}>IMMUTABLE</div>
            </div>
          </div>

          {shards.map((shard) => (
            <div key={shard.id} className={styles.card} style={{'--accent-color': getTypeColor(shard.type)} as any}>
              <div className={styles.cardHeader}>
                <div className={styles.typeIcon}>{getTypeIcon(shard.type)}</div>
                <div className={styles.headerInfo}>
                  <span className={styles.shardName}>{shard.name}</span>
                  <span className={styles.shardTypeLabel} style={{color: getTypeColor(shard.type)}}>{shard.type.replace('_', ' ')}</span>
                </div>
              </div>
              <div className={styles.shardURL}>{shard.db_url}</div>
              <div className={styles.cardFooter}>
                <div className={styles.status}>
                  <div className={styles.statusDot}></div>
                  {shard.status === "active" ? "Connected" : shard.status}
                </div>
                <button 
                  onClick={() => handleDeleteShard(shard.id)} 
                  className={styles.removeBtn}
                  title="Decommission Shard"
                >
                  Decommission
                </button>
              </div>
            </div>
          ))}

          {shards.length === 0 && !isLoading && (
            <div className={styles.emptyState}>
              <div className={styles.emptyIcon}>📡</div>
              <p>No additional shards registered in Core Master.</p>
              <span>Add Global Managers or Specialized Shards to expand the fleet.</span>
            </div>
          )}
        </div>
      )}

      {isModalOpen && (
        <div className={styles.modalOverlay}>
          <div className={styles.modal}>
            <div className={styles.modalHeader}>
              <h3 className={styles.modalTitle}>Register New Shard</h3>
              <p>Expand your infrastructure capacity dynamically.</p>
            </div>
            <form onSubmit={handleAddShard}>
              <div className={styles.formGroup}>
                <label>Shard Infrastructure Role</label>
                <div className={styles.typeSelectorGrid}>
                  {[
                    { id: "global_manager", label: "Global Manager", icon: "🌐" },
                    { id: "user", label: "User Shard", icon: "👥" },
                    { id: "auth", label: "Auth Shard", icon: "🔐" },
                    { id: "analytics", label: "Analytics Shard", icon: "📈" }
                  ].map((t) => (
                    <div 
                      key={t.id} 
                      className={`${styles.typeOption} ${formData.type === t.id ? styles.typeOptionActive : ""}`}
                      onClick={() => setFormData({...formData, type: t.id})}
                    >
                      <span className={styles.typeOptionIcon}>{t.icon}</span>
                      <span className={styles.typeOptionLabel}>{t.label}</span>
                    </div>
                  ))}
                </div>
              </div>

              <div className={styles.formGroup}>
                <label>Display Name</label>
                <input 
                  type="text" 
                  placeholder="e.g. Asia-Pacific Primary Shard" 
                  value={formData.name}
                  onChange={(e) => setFormData({...formData, name: e.target.value})}
                  required
                />
              </div>

              <div className={styles.formGroup}>
                <label>Turso DB URL</label>
                <input 
                  type="text" 
                  placeholder="libsql://your-shard.turso.io" 
                  value={formData.db_url}
                  onChange={(e) => setFormData({...formData, db_url: e.target.value})}
                  required
                />
              </div>

              <div className={styles.formGroup}>
                <label>Turso Auth Token</label>
                <input 
                  type="password" 
                  placeholder="Paste secure token here" 
                  value={formData.token}
                  onChange={(e) => setFormData({...formData, token: e.target.value})}
                  required
                />
              </div>
              <div className={styles.modalActions}>
                <button type="button" onClick={() => setIsModalOpen(false)} className={styles.cancelBtn}>Discard</button>
                <button type="submit" className={styles.submitBtn} disabled={isSubmitting}>
                  {isSubmitting ? "Syncing..." : "Connect Shard"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
