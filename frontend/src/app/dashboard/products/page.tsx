"use client";

import React, { useState, useEffect } from "react";
import styles from "./page.module.css";
import { fetchAPI } from "../../../lib/api";
import Link from "next/link";

interface Product {
  id: string;
  name: string;
  slug: string;
  description: string;
}

export default function ProductsFleetPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  
  // New Product State
  // New/Edit Product State
  const [editingProduct, setEditingProduct] = useState<Product | null>(null);
  const [newName, setNewName] = useState("");
  const [newDesc, setNewDesc] = useState("");
  const [creating, setCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [productToDelete, setProductToDelete] = useState<Product | null>(null);
  const [masterKey, setMasterKey] = useState("");
  const [confirmPhrase, setConfirmPhrase] = useState("");

  const loadProducts = async () => {
    setLoading(true);
    try {
      const res = await fetchAPI("/admin/products");
      if (res.success) setProducts(res.products || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadProducts();
  }, []);

  const handleDeleteProduct = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!productToDelete) return;
    
    setCreating(true);
    try {
      const res = await fetchAPI(`/admin/products/${productToDelete.id}/delete`, { 
        method: "POST",
        body: JSON.stringify({ 
          master_key: masterKey,
          confirmation: confirmPhrase
        })
      });
      if (res.success) {
        setShowDeleteModal(false);
        setProductToDelete(null);
        setMasterKey("");
        setConfirmPhrase("");
        loadProducts();
        alert("Infrastructure decommissioned and shards wiped.");
      } else {
        alert("Error: " + res.message);
      }
    } catch (err: any) {
      alert("Error: " + err.message);
    } finally {
      setCreating(false);
    }
  };

  const openDeleteModal = (p: Product) => {
    setProductToDelete(p);
    setShowDeleteModal(true);
  };

  const openEditModal = (p: Product) => {
    setEditingProduct(p);
    setNewName(p.name);
    setNewDesc(p.description);
    setShowCreateModal(true);
  };

  const handleSaveProduct = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    try {
      const endpoint = editingProduct ? `/admin/products/${editingProduct.id}` : "/admin/products";
      const method = editingProduct ? "PUT" : "POST";
      
      const res = await fetchAPI(endpoint, {
        method,
        body: JSON.stringify({ name: newName, description: newDesc })
      });
      if (res.success) {
        setShowCreateModal(false);
        setEditingProduct(null);
        setNewName("");
        setNewDesc("");
        loadProducts();
        alert(editingProduct ? "Product updated successfully." : "Product Created! Database & Cloud Storage provisioned automatically.");
      }
    } catch (err: any) {
      alert("Error: " + err.message);
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div>
          <h1 className={styles.title}>Infrastructure Fleet</h1>
          <p className={styles.subtitle}>Unified cluster management for BandhanNova platform assets.</p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>+ New Deployment</button>
      </div>

      {loading ? (
        <div className={styles.loading}>SYNCING FLEET...</div>
      ) : (
        <div className={styles.grid}>
          {products.map(p => (
            <div key={p.id} className={`glass-panel ${styles.card}`}>
              <div className={styles.cardActions}>
                <button 
                  className={styles.actionBtn} 
                  onClick={() => openEditModal(p)}
                  title="Edit Deployment"
                >
                  ✎
                </button>
                <button 
                  className={`${styles.actionBtn} ${styles.deleteBtn}`} 
                  onClick={() => openDeleteModal(p)}
                  title="Decommission Fleet"
                >
                  🗑
                </button>
              </div>
              
              <Link href={`/dashboard/products/${p.slug}`} className={styles.cardLink}>
                <div className={styles.icon}>
                  {p.slug === "auth" ? "🔐" : 
                   p.slug === "analytics" ? "📈" : 
                   p.slug === "market" ? "💰" : 
                   p.slug === "ai" ? "🤖" : "📦"}
                </div>
                <div className={styles.info}>
                  <h3>{p.name}</h3>
                  <p>{p.description || "No description provided for this cluster."}</p>
                </div>
                <div className={styles.meta}>
                  <span>🗄️ CLUSTER ONLINE</span>
                  <span>☁️ LFS STORAGE READY</span>
                </div>
                <div className={styles.arrow}>→</div>
              </Link>
            </div>
          ))}
        </div>
      )}

      {showCreateModal && (
        <div className={styles.modalOverlay} onClick={(e) => e.target === e.currentTarget && (setShowCreateModal(false), setEditingProduct(null))}>
          <div className={`glass-panel ${styles.modal}`}>
            <h2>{editingProduct ? "Update Deployment" : "Launch New Deployment"}</h2>
            <p>{editingProduct ? "Modify infrastructure metadata for this cluster." : "This will automatically provision sharded PostgreSQL clusters and Hugging Face LFS storage."}</p>
            <form onSubmit={handleSaveProduct}>
              <div className={styles.formGroup}>
                <label>Deployment Name</label>
                <input 
                  type="text" 
                  className={styles.input} 
                  placeholder="e.g. BandhanNova CRM"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  required 
                  autoFocus
                />
              </div>
              <div className={styles.formGroup}>
                <label>Context / Description</label>
                <textarea 
                  className={styles.input} 
                  style={{ minHeight: "100px", resize: "none" }}
                  placeholder="What is this infrastructure for?"
                  value={newDesc}
                  onChange={(e) => setNewDesc(e.target.value)}
                />
              </div>
              <div className={styles.modalActions}>
                <button type="button" className="btn btn-glass" onClick={() => (setShowCreateModal(false), setEditingProduct(null))}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={creating}>
                  {creating ? "Processing..." : editingProduct ? "Save Changes" : "Confirm & Deploy"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
      {showDeleteModal && productToDelete && (
        <div className={styles.modalOverlay} onClick={(e) => e.target === e.currentTarget && (setShowDeleteModal(false), setProductToDelete(null))}>
          <div className={`glass-panel ${styles.modal}`} style={{ maxWidth: "600px" }}>
            <h2 style={{ color: "var(--danger)" }}>☢ Secure Decommission</h2>
            <p>You are about to PERMANENTLY delete <strong>{productToDelete.name}</strong>. This will wipe all linked database shards and decommission cloud storage. This action is irreversible.</p>
            
            <form onSubmit={handleDeleteProduct}>
              <div className={styles.formGroup}>
                <label>System Master Key</label>
                <input 
                  type="password" 
                  className={styles.input} 
                  placeholder="Enter your BandhanNova Master Key"
                  value={masterKey}
                  onChange={(e) => setMasterKey(e.target.value)}
                  required
                />
              </div>
              
              <div className={styles.formGroup}>
                <label>Verification Phrase</label>
                <p style={{ fontSize: "11px", color: "var(--text-muted)", marginBottom: "8px", textTransform: "none" }}>
                  Type: <em>I am Bandhan, to the best of my knowledge, I want to delete this product, named {productToDelete.name}.</em>
                </p>
                <textarea 
                  className={styles.input} 
                  style={{ minHeight: "80px", fontSize: "13px", resize: "none" }}
                  placeholder="Copy and paste the phrase exactly"
                  value={confirmPhrase}
                  onChange={(e) => setConfirmPhrase(e.target.value)}
                  required
                />
              </div>
              
              <div className={styles.modalActions}>
                <button type="button" className="btn btn-glass" onClick={() => (setShowDeleteModal(false), setProductToDelete(null))}>Abort</button>
                <button type="submit" className="btn btn-primary" style={{ background: "var(--danger)" }} disabled={creating}>
                  {creating ? "Wiping Shards..." : "EXECUTE PURGE 💀"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

