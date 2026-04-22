"use client";
import React, { useState, useEffect } from "react";
import styles from "../page.module.css";
import { fetchAPI } from "../../../../../lib/api";
import DatabaseViewer from "./DatabaseViewer";

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
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [isInspectModalOpen, setIsInspectModalOpen] = useState(false);
  const [selectedShard, setSelectedShard] = useState<Shard | null>(null);
  
  const [formData, setFormData] = useState({ name: "", db_url: "", token: "" });
  const [deleteConfirm, setDeleteConfirm] = useState({ masterKey: "", text: "" });
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Inspector State
  const [tables, setTables] = useState<string[]>([]);
  const [inspecting, setInspecting] = useState(false);

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
      } else {
        alert(res.message || "Failed to add shard");
      }
    } catch (err: any) {
      alert("Connection Failed: " + err.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleEditShard = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedShard) return;
    setIsSubmitting(true);
    try {
      const res = await fetchAPI(`/admin/db/update/${selectedShard.id}`, {
        method: "PUT",
        body: JSON.stringify({ ...formData, product_id: product.id })
      });
      if (res.success) {
        setIsEditModalOpen(false);
        loadShards();
      } else {
        alert(res.message || "Update failed");
      }
    } catch (err: any) {
      alert("Update failed: " + err.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleRemove = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedShard) return;
    if (deleteConfirm.text !== "DELETE") {
      alert("Please type DELETE to confirm");
      return;
    }

    setIsSubmitting(true);
    try {
      const res = await fetchAPI(`/admin/db/remove/${selectedShard.id}`, {
        method: "POST",
        body: JSON.stringify({ master_key: deleteConfirm.masterKey })
      });
      if (res.success) {
        setIsDeleteModalOpen(false);
        setDeleteConfirm({ masterKey: "", text: "" });
        loadShards();
      } else {
        alert(res.message || "Removal failed");
      }
    } catch (err: any) {
      alert("Removal failed: " + err.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  const openInspect = async (db: Shard) => {
    setSelectedShard(db);
    setIsInspectModalOpen(true);
    setInspecting(true);
    try {
      const res = await fetchAPI(`/admin/infrastructure/shards/${db.id}/query`, {
        method: "POST",
        body: JSON.stringify({ query: "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'" })
      });
      if (res.success) {
        setTables(res.data.map((t: any) => t.name));
      }
    } catch (err) {
      console.error(err);
    } finally {
      setInspecting(false);
    }
  };

  const openEdit = (db: Shard) => {
    setSelectedShard(db);
    setFormData({ name: db.name, db_url: db.db_url, token: "" });
    setIsEditModalOpen(true);
  };

  const openDelete = (db: Shard) => {
    setSelectedShard(db);
    setIsDeleteModalOpen(true);
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
                  <div className={styles.shardActions}>
                     <button className="btn btn-glass" style={{fontSize: '11px', padding: '6px 12px'}} onClick={() => openEdit(db)}>Edit</button>
                     <button className="btn btn-glass" style={{fontSize: '11px', padding: '6px 12px'}} onClick={() => openInspect(db)}>Inspect</button>
                  </div>
                  <button className="btn btn-glass" style={{fontSize: '11px', padding: '6px 12px', color: 'var(--danger)'}} onClick={() => openDelete(db)}>🗑️</button>
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

      {/* ─── Edit Shard Modal ────────────────────────── */}
      {isEditModalOpen && (
        <div className={styles.modalOverlay}>
          <div className={`glass-panel ${styles.modalContent}`}>
            <div className={styles.modalHeader}>
               <h3>Edit Shard Credentials</h3>
               <p>Update display name or URL for {selectedShard?.name}</p>
            </div>
            <form onSubmit={handleEditShard} className={styles.uploadForm}>
               <div className={styles.field}>
                  <label>Display Name</label>
                  <input 
                    className={styles.confirmInput}
                    type="text" 
                    value={formData.name}
                    onChange={(e) => setFormData({...formData, name: e.target.value})}
                  />
               </div>
               <div className={styles.field}>
                  <label>Database URL</label>
                  <input 
                    className={styles.confirmInput}
                    type="text" 
                    value={formData.db_url}
                    onChange={(e) => setFormData({...formData, db_url: e.target.value})}
                  />
               </div>
               <div className={styles.field}>
                  <label>New Token (Leave blank to keep current)</label>
                  <input 
                    className={styles.confirmInput}
                    type="password" 
                    placeholder="New access token"
                    value={formData.token}
                    onChange={(e) => setFormData({...formData, token: e.target.value})}
                  />
               </div>
               <div className={styles.modalActions}>
                  <button type="button" className={styles.clearBtn} onClick={() => setIsEditModalOpen(false)}>Cancel</button>
                  <button type="submit" className="btn btn-primary" disabled={isSubmitting}>Update Shard</button>
               </div>
            </form>
          </div>
        </div>
      )}

      {/* ─── Delete Confirmation Modal ──────────────── */}
      {isDeleteModalOpen && (
        <div className={styles.modalOverlay}>
          <div className={`glass-panel ${styles.modalContent}`}>
            <div className={styles.modalHeader}>
               <h3 style={{color: 'var(--danger)'}}>Destructive Action</h3>
               <p>You are about to decommission <strong>{selectedShard?.name}</strong>. This cannot be undone.</p>
            </div>
            <form onSubmit={handleRemove} className={styles.uploadForm}>
               <div className={styles.field}>
                  <label>Admin Master Key</label>
                  <input 
                    className={styles.confirmInput}
                    type="password" 
                    placeholder="Enter Master Key"
                    value={deleteConfirm.masterKey}
                    onChange={(e) => setDeleteConfirm({...deleteConfirm, masterKey: e.target.value})}
                    required
                  />
               </div>
               <div className={styles.field}>
                  <label>Type <strong>DELETE</strong> to confirm</label>
                  <input 
                    className={styles.confirmInput}
                    type="text" 
                    placeholder="DELETE"
                    value={deleteConfirm.text}
                    onChange={(e) => setDeleteConfirm({...deleteConfirm, text: e.target.value})}
                    required
                  />
               </div>
               <div className={styles.modalActions}>
                  <button type="button" className={styles.clearBtn} onClick={() => setIsDeleteModalOpen(false)}>Cancel</button>
                  <button type="submit" className="btn btn-primary" style={{background: 'var(--danger)'}} disabled={isSubmitting}>
                    {isSubmitting ? "Processing..." : "Confirm Destruction"}
                  </button>
               </div>
            </form>
          </div>
        </div>
      )}

      {/* ─── Inspector Modal (Legacy replaced by Viewer) ─── */}
      {isInspectModalOpen && selectedShard && (
        <DatabaseViewer 
          shard={selectedShard} 
          onClose={() => setIsInspectModalOpen(false)} 
        />
      )}
    </div>
  );
}
