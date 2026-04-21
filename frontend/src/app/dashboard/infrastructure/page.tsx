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

export default function InfrastructurePage() {
  const [shards, setShards] = useState<Shard[]>([]);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
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
    try {
      const res = await addShard(formData);
      if (res.success) {
        setIsModalOpen(false);
        setFormData({ name: "", type: "global_manager", db_url: "", token: "" });
        loadShards();
      }
    } catch (error) {
      alert("Failed to add shard");
    }
  };

  const handleDeleteShard = async (id: string) => {
    if (!confirm("Are you sure you want to remove this infrastructure shard? This will disconnect it from the gateway.")) return;
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
      <header className={styles.header}>
        <h2 className={styles.title}>Infrastructure Shards</h2>
        <button onClick={() => setIsModalOpen(true)} className={styles.addBtn}>
          + Register Shard
        </button>
      </header>

      {isLoading ? (
        <p>Loading infrastructure...</p>
      ) : (
        <div className={styles.grid}>
          {shards.map((shard) => (
            <div key={shard.id} className={styles.card}>
              <div className={styles.cardHeader}>
                <span className={styles.shardName}>{shard.name || "Unnamed Shard"}</span>
                <span className={styles.shardType}>{shard.type}</span>
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
                >
                  Decommission
                </button>
              </div>
            </div>
          ))}

          {shards.length === 0 && (
            <p className={styles.emptyState}>No additional shards registered in Core Master.</p>
          )}
        </div>
      )}

      {isModalOpen && (
        <div className={styles.modalOverlay}>
          <div className={styles.modal}>
            <h3 className={styles.modalTitle}>Register New Master Shard</h3>
            <form onSubmit={handleAddShard}>
              <div className={styles.formGroup}>
                <label>Display Name</label>
                <input 
                  type="text" 
                  placeholder="e.g. Global Manager 2" 
                  value={formData.name}
                  onChange={(e) => setFormData({...formData, name: e.target.value})}
                  required
                />
              </div>
              <div className={styles.formGroup}>
                <label>Shard Type</label>
                <select 
                  value={formData.type}
                  onChange={(e) => setFormData({...formData, type: e.target.value})}
                >
                  <option value="global_manager">Global Manager</option>
                  <option value="auth">Auth Shard</option>
                  <option value="analytics">Analytics Shard</option>
                  <option value="user">User Shard</option>
                </select>
              </div>
              <div className={styles.formGroup}>
                <label>Turso DB URL</label>
                <input 
                  type="text" 
                  placeholder="libsql://..." 
                  value={formData.db_url}
                  onChange={(e) => setFormData({...formData, db_url: e.target.value})}
                  required
                />
              </div>
              <div className={styles.formGroup}>
                <label>Turso Auth Token</label>
                <input 
                  type="password" 
                  placeholder="eyJhbG..." 
                  value={formData.token}
                  onChange={(e) => setFormData({...formData, token: e.target.value})}
                  required
                />
              </div>
              <div className={styles.modalActions}>
                <button type="button" onClick={() => setIsModalOpen(false)} className={styles.cancelBtn}>Cancel</button>
                <button type="submit" className={styles.submitBtn}>Register Shard</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
