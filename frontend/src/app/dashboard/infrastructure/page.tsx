"use client";

import React, { useState, useEffect } from "react";
import styles from "./page.module.css";
import { getShards, addShard, updateShard, removeShard, queryShard, clearShard } from "../../../lib/api";

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
  const [editingShard, setEditingShard] = useState<Shard | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [formData, setFormData] = useState({
    name: "",
    type: "global_manager",
    db_url: "",
    token: ""
  });

  const [inspectShard, setInspectShard] = useState<Shard | null>(null);
  const [tables, setTables] = useState<any[]>([]);
  const [selectedTable, setSelectedTable] = useState<string>("");
  const [tableData, setTableData] = useState<any[]>([]);
  const [tableCols, setTableCols] = useState<string[]>([]);
  const [isInspectorLoading, setIsInspectorLoading] = useState(false);

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

  const handleOpenInspect = async (shard: Shard) => {
    setInspectShard(shard);
    setIsInspectorLoading(true);
    setTableData([]);
    setSelectedTable("");
    try {
      const res = await queryShard(shard.id, "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'");
      if (res.success) {
        setTables(res.data || []);
      }
    } catch (error) {
      alert("Failed to connect to shard for inspection");
    }
    setIsInspectorLoading(false);
  };

  const loadTableData = async (tableName: string) => {
    setSelectedTable(tableName);
    setIsInspectorLoading(true);
    try {
      const res = await queryShard(inspectShard!.id, `SELECT * FROM ${tableName} LIMIT 100`);
      if (res.success) {
        setTableData(res.data || []);
        setTableCols(res.columns || []);
      }
    } catch (error) {
      alert("Failed to load table data");
    }
    setIsInspectorLoading(false);
  };

  const handleFactoryReset = async () => {
    if (!confirm("WARNING: This will PERMANENTLY DELETE all data in this shard and re-initialize it. Proceed?")) return;
    setIsInspectorLoading(true);
    try {
      const res = await clearShard(inspectShard!.id);
      if (res.success) {
        alert("Shard successfully reset to factory state.");
        handleOpenInspect(inspectShard!); // Refresh tables
      }
    } catch (error) {
      alert("Failed to reset shard");
    }
    setIsInspectorLoading(false);
  };

  const handleOpenAdd = () => {
    setEditingShard(null);
    setFormData({ name: "", type: "global_manager", db_url: "", token: "" });
    setIsModalOpen(true);
  };

  const handleOpenEdit = (shard: Shard) => {
    setEditingShard(shard);
    setFormData({ 
      name: shard.name, 
      type: shard.type, 
      db_url: shard.db_url, 
      token: "" 
    });
    setIsModalOpen(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      let res;
      if (editingShard) {
        res = await updateShard(editingShard.id, formData);
      } else {
        res = await addShard(formData);
      }
      
      if (res.success) {
        setIsModalOpen(false);
        loadShards();
      }
    } catch (error: any) {
      alert(error.message || "Operation failed. Verify credentials.");
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
        <button onClick={handleOpenAdd} className={styles.addBtn}>
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
          {/* Core Master Card */}
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
                <div className={styles.cardActions}>
                  <button onClick={() => handleOpenInspect(shard)} className={styles.inspectBtn}>Inspect</button>
                  <button onClick={() => handleOpenEdit(shard)} className={styles.editBtn}>Edit</button>
                  <button onClick={() => handleDeleteShard(shard.id)} className={styles.removeBtn}>Remove</button>
                </div>
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

      {/* Inspector Modal */}
      {inspectShard && (
        <div className={styles.modalOverlay}>
          <div className={`${styles.modal} ${styles.inspectorModal}`}>
            <div className={styles.modalHeader}>
              <div className={styles.headerTop}>
                <h3 className={styles.modalTitle}>Shard Explorer: {inspectShard.name}</h3>
                <button onClick={() => setInspectShard(null)} className={styles.closeBtn}>×</button>
              </div>
              <div className={styles.inspectorActions}>
                <div className={styles.tableSelector}>
                  {tables.map(t => (
                    <button 
                      key={t.name} 
                      onClick={() => loadTableData(t.name)}
                      className={`${styles.tableTag} ${selectedTable === t.name ? styles.tableTagActive : ""}`}
                    >
                      {t.name}
                    </button>
                  ))}
                  {tables.length === 0 && <span className={styles.noTables}>No tables found</span>}
                </div>
                <button onClick={handleFactoryReset} className={styles.resetBtn}>Factory Reset</button>
              </div>
            </div>
            
            <div className={styles.inspectorBody}>
              {isInspectorLoading ? (
                <div className={styles.innerLoading}><div className={styles.spinner}></div></div>
              ) : selectedTable ? (
                <div className={styles.tableWrapper}>
                  <table className={styles.dataTable}>
                    <thead>
                      <tr>
                        {tableCols.map(col => <th key={col}>{col}</th>)}
                      </tr>
                    </thead>
                    <tbody>
                      {tableData.map((row, i) => (
                        <tr key={i}>
                          {tableCols.map(col => <td key={col}>{String(row[col])}</td>)}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  {tableData.length === 0 && <p className={styles.noData}>No rows in this table</p>}
                </div>
              ) : (
                <div className={styles.welcomeInspector}>
                  <div className={styles.welcomeIcon}>🔍</div>
                  <p>Select a table to browse data on {inspectShard.name}</p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {isModalOpen && (
        <div className={styles.modalOverlay}>
          <div className={styles.modal}>
            <div className={styles.modalHeader}>
              <h3 className={styles.modalTitle}>{editingShard ? "Update Shard" : "Register New Shard"}</h3>
              <p>{editingShard ? "Modify existing infrastructure configuration." : "Expand your infrastructure capacity dynamically."}</p>
            </div>
            <form onSubmit={handleSubmit}>
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
                <label>Turso Auth Token {editingShard && <span style={{fontSize: '0.7rem', color: '#666'}}>(Optional - leave blank to keep current)</span>}</label>
                <input 
                  type="password" 
                  placeholder={editingShard ? "••••••••••••" : "Paste secure token here"} 
                  value={formData.token}
                  onChange={(e) => setFormData({...formData, token: e.target.value})}
                  required={!editingShard}
                />
              </div>
              <div className={styles.modalActions}>
                <button type="button" onClick={() => setIsModalOpen(false)} className={styles.cancelBtn}>Discard</button>
                <button type="submit" className={styles.submitBtn} disabled={isSubmitting}>
                  {isSubmitting ? "Syncing..." : editingShard ? "Update Shard" : "Connect Shard"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
