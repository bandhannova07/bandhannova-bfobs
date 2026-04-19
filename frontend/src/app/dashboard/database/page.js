"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import styles from "./page.module.css";
import { fetchAPI } from "../../../lib/api";
import PulseHealth from "../../../components/PulseHealth";

export default function DatabasePage() {
  const [databases, setDatabases] = useState([]);
  const [products, setProducts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedProduct, setSelectedProduct] = useState(null);
  const [search, setSearch] = useState("");
  
  const [showModal, setShowModal] = useState(false);
  const [showProductModal, setShowProductModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [editingProduct, setEditingProduct] = useState(null);
  const [showSecret, setShowSecret] = useState(false);
  
  const [formData, setFormData] = useState({ name: "", category: "user", db_url: "", token: "", product_id: "" });
  const [productFormData, setProductFormData] = useState({ name: "", app_type: "website", app_url: "", description: "", icon: "" });
  const [deleteData, setDeleteData] = useState({ master_key: "", confirmation: "" });
  
  const [addLoading, setAddLoading] = useState(false);
  const [error, setError] = useState("");

  const loadData = async () => {
    setLoading(true);
    try {
      const [dbRes, prodRes] = await Promise.all([
        fetchAPI("/admin/databases"),
        fetchAPI("/admin/products")
      ]);
      
      if (dbRes.success) setDatabases(dbRes.databases || []);
      if (prodRes.success) setProducts(prodRes.products || []);
    } catch (err) {
      console.error("Failed to load data", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleAddDB = async (e) => {
    e.preventDefault();
    setAddLoading(true);
    setError("");
    try {
      const res = await fetchAPI("/admin/databases", {
        method: "POST",
        body: JSON.stringify(formData),
      });
      if (res.success) {
        setShowModal(false);
        setFormData({ name: "", category: "user", db_url: "", token: "", product_id: selectedProduct?.id || "" });
        loadData();
      }
    } catch (err) {
      setError(err.message || "Failed to add database");
    } finally {
      setAddLoading(false);
    }
  };

  const handleSaveProduct = async (e) => {
    e.preventDefault();
    setAddLoading(true);
    setError("");
    try {
      const url = editingProduct ? `/admin/products/${editingProduct.id}` : "/admin/products";
      const method = editingProduct ? "PUT" : "POST";
      
      const res = await fetchAPI(url, {
        method: method,
        body: JSON.stringify(productFormData),
      });
      if (res.success) {
        setShowProductModal(false);
        setEditingProduct(null);
        setProductFormData({ name: "", app_type: "website", app_url: "", description: "", icon: "" });
        loadData();
      }
    } catch (err) {
      setError(err.message || "Failed to save product");
    } finally {
      setAddLoading(false);
    }
  };

  const handleResetOAuth = async () => {
    if (!confirm("Are you sure? This will invalidate existing connections for this product.")) return;
    try {
      const res = await fetchAPI(`/admin/products/${selectedProduct.id}/reset-oauth`, {
        method: "POST"
      });
      if (res.success) {
        loadData();
        // Since selectedProduct is stale, we need to update it or close view
        setSelectedProduct(null); 
      }
    } catch (err) {
      alert("Failed to reset credentials");
    }
  };

  const handleDeleteProduct = async (e) => {
    e.preventDefault();
    setAddLoading(true);
    setError("");
    try {
      const res = await fetchAPI(`/admin/products/${selectedProduct.id}/delete`, {
        method: "POST",
        body: JSON.stringify(deleteData),
      });
      if (res.success) {
        setShowDeleteModal(false);
        setSelectedProduct(null);
        setDeleteData({ master_key: "", confirmation: "" });
        loadData();
      }
    } catch (err) {
      setError(err.message || "Failed to delete product");
    } finally {
      setAddLoading(false);
    }
  };

  const getProductStats = (productId) => {
    const pDbs = databases.filter(db => db.product_id === productId);
    return {
      count: pDbs.length,
      status: pDbs.length === 0 ? "Empty" : (pDbs.every(db => db.status === "active") ? "Healthy" : "Attention Required")
    };
  };

  const coreDBs = databases.filter(db => db.is_core);
  const unusedDBs = databases.filter(db => !db.is_core && !db.product_id);
  
  const filteredDBs = databases.filter(db => {
    const matchesSearch = db.name.toLowerCase().includes(search.toLowerCase()) || 
                         db.db_url.toLowerCase().includes(search.toLowerCase());
    
    if (selectedProduct) {
      return matchesSearch && db.product_id === selectedProduct.id;
    }
    return false;
  });

  return (
    <div>
      <div className={styles.header}>
        <div>
          <h2 style={{ fontSize: "24px", fontWeight: "700", marginBottom: "8px" }}>
            {selectedProduct ? selectedProduct.name : "BandhanNova Ecosystem"}
          </h2>
          <div className={styles.breadcrumb}>
            <span className={styles.breadcrumbItem} onClick={() => setSelectedProduct(null)}>Fleet</span>
            {selectedProduct && (
              <>
                <span className={styles.breadcrumbSeparator}>/</span>
                <span className={styles.breadcrumbItem}>{selectedProduct.name}</span>
              </>
            )}
          </div>
        </div>
        <div className={styles.headerActions}>
          <input 
            type="text" 
            placeholder="Search assets..." 
            className={styles.searchBar}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          {selectedProduct ? (
            <div style={{ display: "flex", gap: "12px" }}>
              {selectedProduct.id !== "core" && (
                <>
                  <button className="btn btn-glass" onClick={() => {
                    setEditingProduct(selectedProduct);
                    setProductFormData({
                      name: selectedProduct.name,
                      app_type: selectedProduct.app_type || "website",
                      app_url: selectedProduct.app_url || "",
                      description: selectedProduct.description || "",
                      icon: selectedProduct.icon || ""
                    });
                    setShowProductModal(true);
                  }}>Edit</button>
                  <button className="btn btn-danger" onClick={() => setShowDeleteModal(true)}>Delete</button>
                </>
              )}
              <button className="btn btn-primary" onClick={() => {
                const count = databases.filter(db => db.category === "user" && db.product_id === (selectedProduct.id === "core" ? "" : selectedProduct.id)).length;
                setFormData({ 
                  name: `User Shard ${count}`, 
                  category: "user", 
                  db_url: "", 
                  token: "", 
                  product_id: selectedProduct.id === "core" ? "" : selectedProduct.id 
                });
                setShowModal(true);
              }}>
                + Add Shard
              </button>
            </div>
          ) : (
            <button className="btn btn-primary" onClick={() => {
              setEditingProduct(null);
              setProductFormData({ name: "", app_type: "website", app_url: "", description: "", icon: "" });
              setShowProductModal(true);
            }}>
              + New Product
            </button>
          )}
        </div>
      </div>
      
      {!selectedProduct && <PulseHealth />}
      
      {loading ? (
        <div style={{ padding: "40px", textAlign: "center" }}>Scanning Ecosystem Topology...</div>
      ) : (
        <>
          {!selectedProduct ? (
            <div className={styles.grid}>
              {/* System Core Card (Always present) */}
              <div className={styles.productCard} onClick={() => setSelectedProduct({ id: "core", name: "System Core" })}>
                <div className={styles.productIcon}>🛡️</div>
                <div className={styles.productName}>System Core</div>
                <div className={styles.productDesc}>Vital infrastructure shards and unused database resources.</div>
                <div className={styles.productMeta}>
                  <span>{coreDBs.length + unusedDBs.length} Total Shards</span>
                  <span style={{ color: "var(--neon-green)" }}>Critical</span>
                </div>
              </div>

              {products.map((product) => {
                const stats = getProductStats(product.id);
                return (
                  <div key={product.id} className={styles.productCard} onClick={() => setSelectedProduct(product)}>
                    <div className={styles.productIcon}>
                      {product.icon ? (
                        <img src={product.icon} alt={product.name} style={{ width: "100%", height: "100%", borderRadius: "8px", objectFit: "cover" }} />
                      ) : (
                        "📦"
                      )}
                    </div>
                    <div className={styles.productName}>{product.name}</div>
                    <div className={styles.productDesc}>{product.description || "No description provided."}</div>
                    <div className={styles.productMeta}>
                      <span>{stats.count} Shards</span>
                      <span style={{ color: stats.status === "Healthy" ? "var(--neon-green)" : (stats.status === "Empty" ? "var(--text-secondary)" : "var(--neon-amber)") }}>
                        {stats.status}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <div>
              {selectedProduct.id === "core" && unusedDBs.length > 0 && (
                <div style={{ marginBottom: "32px" }}>
                  <h3 style={{ fontSize: "14px", color: "var(--neon-amber)", textTransform: "uppercase", letterSpacing: "1px", marginBottom: "16px" }}>Unused / Orphaned Shards</h3>
                  <div className={styles.grid}>
                    {unusedDBs.map((db) => (
                      <Link href={`/dashboard/database/${db.slug}`} key={db.id} style={{textDecoration: "none", color: "inherit", display: "block"}}>
                        <div className={styles.shardCard} style={{ borderStyle: "dashed", borderColor: "var(--neon-amber)" }}>
                          <div className={styles.shardTop}>
                            <div>
                              <div className={styles.shardType} style={{ background: "rgba(255, 184, 0, 0.1)", color: "var(--neon-amber)" }}>ORPHANED</div>
                              <div className={styles.shardName}>{db.name}</div>
                            </div>
                            <div className="badge badge-offline">AVAILABLE</div>
                          </div>
                          <div style={{ fontSize: "12px", color: "var(--text-secondary)" }}>URL: {db.db_url}</div>
                        </div>
                      </Link>
                    ))}
                  </div>
                  <div style={{ height: "1px", background: "var(--glass-border)", margin: "32px 0" }} />
                </div>
              )}

              <div className={styles.grid}>
                {selectedProduct.id !== "core" && (
                  <div className={styles.shardCard} style={{ gridColumn: "1 / -1", border: "1px solid var(--neon-purple)", background: "rgba(139, 92, 246, 0.05)" }}>
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "16px" }}>
                      <h3 style={{ fontSize: "14px", color: "var(--neon-purple)", textTransform: "uppercase" }}>OAuth 2.0 Credentials (BandhanNova ID)</h3>
                      <div style={{ display: "flex", gap: "8px" }}>
                        <button className="btn btn-glass" style={{ padding: "4px 12px", fontSize: "11px" }} onClick={handleResetOAuth}>
                          {selectedProduct.client_id ? "Reset Credentials" : "Generate Credentials"}
                        </button>
                        <span className="badge" style={{ background: "rgba(139, 92, 246, 0.2)", color: "var(--neon-purple)" }}>OIDC COMPLIANT</span>
                      </div>
                    </div>
                    <div style={{ display: "flex", flexDirection: "column", gap: "12px" }}>
                      <div style={{ display: "flex", gap: "12px" }}>
                        <div style={{ flex: 1 }}>
                          <label style={{ fontSize: "10px", color: "var(--text-secondary)", display: "block", marginBottom: "4px" }}>CLIENT ID</label>
                          <code style={{ background: "rgba(0,0,0,0.3)", padding: "8px", borderRadius: "4px", fontSize: "12px", display: "block", color: "var(--neon-blue)" }}>
                            {selectedProduct.client_id || "NOT_ASSIGNED"}
                          </code>
                        </div>
                        <div style={{ flex: 1 }}>
                          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "4px" }}>
                            <label style={{ fontSize: "10px", color: "var(--text-secondary)" }}>CLIENT SECRET</label>
                            <button 
                              style={{ fontSize: "10px", color: "var(--neon-blue)", padding: 0 }}
                              onClick={() => setShowSecret(!showSecret)}
                            >
                              {showSecret ? "Hide" : "Show"}
                            </button>
                          </div>
                          <code style={{ background: "rgba(0,0,0,0.3)", padding: "8px", borderRadius: "4px", fontSize: "12px", display: "block", wordBreak: "break-all" }}>
                            {selectedProduct.client_secret ? (showSecret ? selectedProduct.client_secret : "••••••••••••••••••••••••••••••••") : "NOT_ASSIGNED"}
                          </code>
                        </div>
                      </div>
                      <div style={{ fontSize: "11px", color: "var(--text-secondary)" }}>
                        {selectedProduct.client_id ? 
                          'Use these credentials to integrate "Continue with BandhanNova" button in this product.' : 
                          'Generate credentials to enable BandhanNova ID for this product.'}
                      </div>
                    </div>
                  </div>
                )}
                
                {(selectedProduct.id === "core" ? coreDBs : filteredDBs).map((db) => (
                  <Link href={`/dashboard/database/${db.slug}`} key={db.id} style={{textDecoration: "none", color: "inherit", display: "block"}}>
                    <div className={styles.shardCard} style={{ cursor: "pointer", transition: "transform 0.2s" }} onMouseEnter={(e) => e.currentTarget.style.transform = "scale(1.02)"} onMouseLeave={(e) => e.currentTarget.style.transform = "scale(1)"}>
                      <div className={styles.shardTop}>
                        <div>
                          <div className={`${styles.shardType} ${styles[`type-${db.category}`]}`}>
                            {db.category} Shard
                          </div>
                          <div className={styles.shardName}>{db.name}</div>
                        </div>
                        <div className={`${styles.healthBadge} badge badge-${db.status.toLowerCase()}`}>
                          {db.status}
                        </div>
                      </div>

                      <div style={{ marginTop: "16px", fontSize: "12px", color: "var(--text-secondary)", wordBreak: "break-all" }}>
                        <strong>URL:</strong> {db.db_url}
                      </div>
                      
                      <div style={{ marginTop: "12px", fontSize: "11px", display: "flex", gap: "8px" }}>
                        {db.is_core ? (
                          <span className="badge badge-active">Immutable Core</span>
                        ) : (
                          <span className="badge badge-healthy">Active Resource</span>
                        )}
                      </div>
                    </div>
                  </Link>
                ))}

                {(selectedProduct.id === "core" ? coreDBs : filteredDBs).length === 0 && (
                  <div style={{ padding: "40px", color: "var(--text-secondary)" }}>
                    No active shards found.
                  </div>
                )}
              </div>
            </div>
          )}
        </>
      )}

      {/* Add/Edit Product Modal */}
      {showProductModal && (
        <div className={styles.modalOverlay} onClick={(e) => e.target === e.currentTarget && setShowProductModal(false)}>
          <div className={styles.modal}>
            <div className={styles.modalHeader}>
              <span>{editingProduct ? "Modify Product Details" : "Launch New Product"}</span>
              <button className={styles.closeBtn} onClick={() => setShowProductModal(false)}>×</button>
            </div>
            <form onSubmit={handleSaveProduct} style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
              <div style={{ display: "flex", gap: "24px", alignItems: "center" }}>
                <div className={styles.productIcon} style={{ margin: 0 }}>
                  {productFormData.icon ? (
                    <img src={productFormData.icon} alt="Preview" style={{ width: "100%", height: "100%", borderRadius: "8px", objectFit: "cover" }} />
                  ) : (
                    "📦"
                  )}
                </div>
                <div className={styles.formGroup} style={{ flex: 1 }}>
                  <label>Icon URL</label>
                  <input type="text" className={styles.input} value={productFormData.icon} onChange={(e) => setProductFormData({...productFormData, icon: e.target.value})} placeholder="https://.../logo.png" />
                </div>
              </div>
              <div className={styles.formGroup}>
                <label>Application Type</label>
                <select className={styles.select} value={productFormData.app_type} onChange={(e) => setProductFormData({...productFormData, app_type: e.target.value})}>
                  <option value="website">Web Application / Portal</option>
                  <option value="mobile">Mobile App (Android/iOS)</option>
                  <option value="api">Internal API Service</option>
                </select>
              </div>
              <div className={styles.formGroup}>
                <label>Product Name</label>
                <input type="text" className={styles.input} value={productFormData.name} onChange={(e) => setProductFormData({...productFormData, name: e.target.value})} placeholder="e.g. BandhanNova Chat" required />
              </div>
              <div className={styles.formGroup}>
                <label>Platform URL</label>
                <input type="url" className={styles.input} value={productFormData.app_url} onChange={(e) => setProductFormData({...productFormData, app_url: e.target.value})} placeholder="https://..." />
              </div>
              <div className={styles.formGroup}>
                <label>Details</label>
                <textarea className={styles.input} style={{ minHeight: "80px", resize: "none" }} value={productFormData.description} onChange={(e) => setProductFormData({...productFormData, description: e.target.value})} placeholder="Product purpose..." />
              </div>
              {error && <div className={styles.errorText}>{error}</div>}
              <div className={styles.modalFooter}>
                <button type="button" className="btn btn-glass" onClick={() => setShowProductModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={addLoading}>{addLoading ? "Processing..." : (editingProduct ? "Save Changes" : "Launch Product")}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Delete Product Modal */}
      {showDeleteModal && (
        <div className={styles.modalOverlay} onClick={(e) => e.target === e.currentTarget && setShowDeleteModal(false)}>
          <div className={styles.modal} style={{ borderColor: "var(--neon-red)" }}>
            <div className={styles.modalHeader} style={{ color: "var(--neon-red)" }}>
              <span>CRITICAL: PERMANENT DELETION</span>
              <button className={styles.closeBtn} onClick={() => setShowDeleteModal(false)}>×</button>
            </div>
            <p style={{ fontSize: "14px", color: "var(--text-secondary)", lineHeight: "1.6" }}>
              Warning: This action will permanently delete <strong>{selectedProduct.name}</strong> and <strong>WIPE ALL DATA</strong> from its connected shards. The shards will be moved to the Unused pool.
            </p>
            <form onSubmit={handleDeleteProduct} style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
              <div className={styles.formGroup}>
                <label>BandhanNova Master Key</label>
                <input type="password" className={styles.input} value={deleteData.master_key} onChange={(e) => setDeleteData({...deleteData, master_key: e.target.value})} required />
              </div>
              <div className={styles.formGroup}>
                <label>Retype Confirmation Phrase</label>
                <div 
                  style={{ 
                    fontSize: "11px", 
                    color: "var(--neon-amber)", 
                    marginBottom: "4px",
                    userSelect: "none",
                    WebkitUserSelect: "none",
                    msUserSelect: "none",
                    cursor: "default"
                  }}
                  onContextMenu={(e) => e.preventDefault()}
                  onCopy={(e) => {
                    e.preventDefault();
                    return false;
                  }}
                >
                  I am Bandhan, to the best of my knowledge, I want to delete this product, named {selectedProduct.name}.
                </div>
                <input 
                  type="text" 
                  className={styles.input} 
                  value={deleteData.confirmation} 
                  onChange={(e) => setDeleteData({...deleteData, confirmation: e.target.value})} 
                  placeholder="Type exactly..." 
                  autoComplete="off"
                  onPaste={(e) => {
                    e.preventDefault();
                    return false;
                  }}
                  required 
                />
              </div>
              {error && <div className={styles.errorText}>{error}</div>}
              <div className={styles.modalFooter}>
                <button type="button" className="btn btn-glass" onClick={() => setShowDeleteModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-danger" disabled={addLoading}>{addLoading ? "Wiping Data..." : "Confirm & Wipe Everything"}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Add Database Modal */}
      {showModal && (
        <div className={styles.modalOverlay} onClick={(e) => e.target === e.currentTarget && setShowModal(false)}>
          <div className={styles.modal}>
            <div className={styles.modalHeader}>
              <span>Connect New Shard</span>
              <button className={styles.closeBtn} onClick={() => setShowModal(false)}>×</button>
            </div>
            <form onSubmit={handleAddDB} style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
              <div className={styles.formGroup}>
                <label>Parent Product (Optional)</label>
                <select 
                  className={styles.select} 
                  value={formData.product_id} 
                  onChange={(e) => setFormData({...formData, product_id: e.target.value})}
                >
                  <option value="">None (Unused Pool)</option>
                  {products.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
                </select>
              </div>
              <div className={styles.formGroup}>
                <label>Category</label>
                <select 
                  className={styles.select} 
                  value={formData.category} 
                  onChange={(e) => {
                    const cat = e.target.value;
                    const count = databases.filter(db => db.category === cat && !db.is_core).length;
                    const baseNames = { user: "User Shard", auth: "Auth Shard", analytics: "Analytics Shard", global: "Global Manager" };
                    setFormData({...formData, category: cat, name: `${baseNames[cat]} ${count}`});
                  }}
                >
                  <option value="user">User Data Shard</option>
                  <option value="auth">Authentication Shard</option>
                  <option value="analytics">Analytics/Log Shard</option>
                </select>
              </div>
              <div className={styles.formGroup}>
                <label>Shard Name</label>
                <input type="text" className={styles.input} value={formData.name} onChange={(e) => setFormData({...formData, name: e.target.value})} placeholder="e.g. EU Region Users" required />
              </div>
              <div className={styles.formGroup}>
                <label>Turso URL</label>
                <input type="text" className={styles.input} value={formData.db_url} onChange={(e) => setFormData({...formData, db_url: e.target.value})} placeholder="libsql://..." required />
              </div>
              <div className={styles.formGroup}>
                <label>Turso Auth Token</label>
                <input type="password" className={styles.input} value={formData.token} onChange={(e) => setFormData({...formData, token: e.target.value})} placeholder="Secret token" required />
              </div>
              {error && <div className={styles.errorText}>{error}</div>}
              <div className={styles.modalFooter}>
                <button type="button" className="btn btn-glass" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={addLoading}>{addLoading ? "Connecting..." : "Initialize Shard"}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
