"use client";
import React, { useState, useEffect } from "react";
import styles from "../page.module.css";
import { fetchAPI } from "../../../../../lib/api";

interface Shard {
  id: string;
  name: string;
  db_url: string;
  status: string;
}

interface Product {
  id: string;
  slug: string;
}

interface DatabaseViewProps {
  product: Product;
}

export default function DatabaseView({ product }: DatabaseViewProps) {
  const [shards, setShards] = useState<Shard[]>([]);
  const [loading, setLoading] = useState(true);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  const [formData, setFormData] = useState({
    name: "",
    db_url: "",
    token: ""
  });

  const loadShards = async () => {
    setLoading(true);
    try {
      const res = await fetchAPI(`/admin/databases?product_id=${product.id}`);
      if (res.success) setShards(res.databases || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadShards();
  }, [product.slug]);

  const handleAddShard = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      const res = await fetchAPI("/admin/db/provision", {
        method: "POST",
        body: JSON.stringify({ 
          ...formData, 
          product_id: product.id, 
          category: "user" 
        })
      });
      if (res.success) {
        setIsModalOpen(false);
        setFormData({ name: "", db_url: "", token: "" });
        loadShards();
      }
    } catch (err: any) {
      alert("Connection Failed: " + err.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleRemove = async (id: string) => {
    if (!confirm("Are you sure you want to PERMANENTLY decommission this shard?")) return;
    try {
      const res = await fetchAPI(`/admin/db/remove/${id}`, { method: "POST" });
      if (res.success) {
        loadShards();
      }
    } catch (err: any) {
      alert("Error: " + err.message);
    }
  };

  return (
    <div className={styles.tabContent}>
      <div className={styles.sectionHeader}>
        <h3 className={styles.sectionTitle}>Infrastructure Fleet</h3>
        <button className="btn btn-primary" onClick={() => setIsModalOpen(true)}>+ Add Dedicated Shard</button>
      </div>

      {loading ? (
        <div className={styles.loading}>SYNCING FLEET...</div>
      ) : shards.length === 0 ? (
        <div className={styles.emptyState}>
           <div className={styles.emptyIcon}>📡</div>
           <p>No dedicated shards assigned to this product.</p>
           <button className="btn btn-primary" style={{marginTop: '20px'}} onClick={() => setIsModalOpen(true)}>Link First Shard</button>
        </div>
      ) : (
        <div className={styles.dbGrid}>
          {shards.map(db => (
            <div key={db.id} className={`glass-panel ${styles.shardCard}`}>
               <div className={styles.shardHeader}>
                  <div className={styles.shardIcon}>🗄️</div>
                  <div className={styles.shardMeta}>
                     <h4>{db.name}</h4>
                     <span className={styles.shardStatusLabel}>ACTIVE SHARD</span>
                  </div>
               </div>
               <code className={styles.shardURL}>{db.db_url}</code>
               <div className={styles.shardFooter}>
                  <div className={styles.shardStatus}>
                     <div className={styles.statusDot}></div>
                     CONNECTED
                  </div>
                  <button className="btn btn-glass" style={{fontSize: '11px', padding: '6px 12px'}} onClick={() => handleRemove(db.id)}>Decommission</button>
               </div>
            </div>
          ))}
        </div>
      )}

      {/* ─── Add Shard Modal ─────────────────────────── */}
      {isModalOpen && (
        <div className={styles.modalOverlay}>
          <div className={`glass-panel ${styles.modalContent}`}>
            <div className={styles.modalHeader}>
               <h3>Register Dedicated Shard</h3>
               <p>Connect a pre-created Turso database to this product fleet.</p>
            </div>
            <form onSubmit={handleAddShard} className={styles.uploadForm}>
               <div className={styles.field}>
                  <label>Display Name</label>
                  <input 
                    className={styles.confirmInput}
                    type="text" 
                    placeholder="e.g. Primary Data Shard"
                    value={formData.name}
                    onChange={(e) => setFormData({...formData, name: e.target.value})}
                    required
                  />
               </div>
               <div className={styles.field}>
                  <label>Database URL</label>
                  <input 
                    className={styles.confirmInput}
                    type="text" 
                    placeholder="libsql://your-db.turso.io"
                    value={formData.db_url}
                    onChange={(e) => setFormData({...formData, db_url: e.target.value})}
                    required
                  />
               </div>
               <div className={styles.field}>
                  <label>Access Token</label>
                  <input 
                    className={styles.confirmInput}
                    type="password" 
                    placeholder="Paste secure token here"
                    value={formData.token}
                    onChange={(e) => setFormData({...formData, token: e.target.value})}
                    required
                  />
               </div>
               <div className={styles.modalActions}>
                  <button type="button" className={styles.clearBtn} onClick={() => setIsModalOpen(false)}>Cancel</button>
                  <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
                    {isSubmitting ? "Verifying..." : "Link Shard"}
                  </button>
               </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

