"use client";

import { useState, useEffect } from "react";
import styles from "./page.module.css";
import { fetchAPI } from "../../../lib/api";
import PulseHealth from "../../../components/PulseHealth";

export default function DatabaseLabPage() {
  const [view, setView] = useState("fleet"); // fleet, product
  const [products, setProducts] = useState([]);
  const [selectedProduct, setSelectedProduct] = useState(null);
  const [activeTab, setActiveTab] = useState("sql"); // sql, tables, shards
  
  const [shards, setShards] = useState([]);
  const [loading, setLoading] = useState(true);
  
  // SQL Editor State
  const [sqlQuery, setSqlQuery] = useState("SELECT * FROM users LIMIT 10;");
  const [targetShards, setTargetShards] = useState([]); // Array of slugs
  const [bulkResults, setBulkResults] = useState(null);
  const [queryLoading, setQueryLoading] = useState(false);
  const [saveToMaster, setSaveToMaster] = useState(true);

  // New Shard State
  const [showProvisionModal, setShowProvisionModal] = useState(false);
  const [newShardName, setNewShardName] = useState("");
  const [provisioning, setProvisioning] = useState(false);

  useEffect(() => {
    loadInitialData();
  }, []);

  const loadInitialData = async () => {
    setLoading(true);
    try {
      const [prodRes, shardRes] = await Promise.all([
        fetchAPI("/admin/products"),
        fetchAPI("/admin/db/status")
      ]);
      if (prodRes.success) setProducts(prodRes.products);
      if (shardRes.success) setShards(shardRes.shards);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const openProduct = (product) => {
    setSelectedProduct(product);
    setSqlQuery(product.master_schema || "/* Define your product schema here */\nCREATE TABLE IF NOT EXISTS users (\n  id TEXT PRIMARY KEY,\n  email TEXT UNIQUE,\n  created_at INTEGER\n);");
    
    // Auto-select all shards for this product
    const productShards = shards.filter(s => s.product_id === product.id).map(s => s.name);
    setTargetShards(productShards);
    
    setView("product");
    setActiveTab("sql");
  };

  const handleBulkExecute = async () => {
    if (targetShards.length === 0) {
      alert("Please select at least one database to execute.");
      return;
    }
    setQueryLoading(true);
    setBulkResults(null);
    try {
      const res = await fetchAPI("/admin/db/execute-bulk", {
        method: "POST",
        body: JSON.stringify({
          product_id: selectedProduct.id,
          shard_slugs: targetShards,
          sql: sqlQuery,
          save_to_master: saveToMaster
        }),
      });
      if (res.success) {
        setBulkResults(res.results);
      }
    } catch (err) {
      alert("Execution failed: " + err.message);
    } finally {
      setQueryLoading(false);
    }
  };

  const handleProvisionShard = async (e) => {
    e.preventDefault();
    setProvisioning(true);
    try {
      const res = await fetchAPI("/admin/db/provision", {
        method: "POST",
        body: JSON.stringify({
          name: newShardName,
          category: "user",
          product_id: selectedProduct.id
        }),
      });
      if (res.success) {
        setShowProvisionModal(false);
        setNewShardName("");
        await loadInitialData(); // Refresh list
        alert("New Shard Provisioned and Schema Synced!");
      }
    } catch (err) {
      alert("Provisioning failed: " + err.message);
    } finally {
      setProvisioning(false);
    }
  };

  const toggleShard = (slug) => {
    setTargetShards(prev => 
      prev.includes(slug) ? prev.filter(s => s !== slug) : [...prev, slug]
    );
  };

  if (view === "fleet") {
    return (
      <div className={styles.container}>
        <div className={styles.header}>
          <div>
            <h1 className={styles.title}>BandhanNova DB Fleet</h1>
            <p className={styles.subtitle}>Select a product card to manage its database ecosystem.</p>
          </div>
        </div>
        <PulseHealth />
        {loading ? (
          <div className={styles.loading}>Scanning Fleet...</div>
        ) : (
          <div className={styles.productGrid}>
            {products.map(p => (
              <div key={p.id} className={styles.productCard} onClick={() => openProduct(p)}>
                <div className={styles.productIcon}>{p.icon || "📦"}</div>
                <h3>{p.name}</h3>
                <p>{p.description || "No description."}</p>
                <div className={styles.productFooter}>
                  <span>{shards.filter(s => s.product_id === p.id).length} Shards</span>
                  <div className={styles.dbUrl}>
                    <code>bn-db://{p.slug}</code>
                  </div>
                </div>
              </div>
            ))}
            <div className={styles.addProductCard} onClick={() => alert("Redirecting to Product Manager...")}>
              <span>+ Add New Product</span>
            </div>
          </div>
        )}
      </div>
    );
  }

  // Product Dedicated View
  const productShards = shards.filter(s => s.product_id === selectedProduct.id);

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div className={styles.breadcrumb}>
          <span onClick={() => setView("fleet")} style={{ cursor: "pointer", color: "var(--neon-blue)" }}>Fleet</span>
          <span>/</span>
          <span>{selectedProduct.name}</span>
        </div>
        <div className={styles.tabs}>
          <button className={`${styles.tab} ${activeTab === "sql" ? styles.activeTab : ""}`} onClick={() => setActiveTab("sql")}>SQL Editor</button>
          <button className={`${styles.tab} ${activeTab === "tables" ? styles.activeTab : ""}`} onClick={() => setActiveTab("tables")}>Tables</button>
          <button className={`${styles.tab} ${activeTab === "shards" ? styles.activeTab : ""}`} onClick={() => setActiveTab("shards")}>Databases</button>
        </div>
      </div>

      <div className={styles.productContent}>
        {activeTab === "sql" && (
          <div className={styles.sqlLayout}>
            <div className={styles.sqlMain}>
              <div className={styles.editorHeader}>
                <h3>Product Master Schema</h3>
                <div className={styles.editorActions}>
                  <label className={styles.checkbox}>
                    <input type="checkbox" checked={saveToMaster} onChange={(e) => setSaveToMaster(e.target.checked)} />
                    Save to Master SQL
                  </label>
                  <button className="btn btn-primary" onClick={handleBulkExecute} disabled={queryLoading}>
                    {queryLoading ? "Executing..." : "Execute on Selected"}
                  </button>
                </div>
              </div>
              <textarea 
                className={styles.sqlEditor}
                value={sqlQuery}
                onChange={(e) => setSqlQuery(e.target.value)}
              />
              {bulkResults && (
                <div className={styles.bulkResults}>
                  {Object.entries(bulkResults).map(([slug, res]) => (
                    <div key={slug} className={styles.shardResult}>
                      <div className={styles.shardResultHeader}>
                        <span>{slug}</span>
                        <span className={res.success ? styles.successText : styles.errorText}>
                          {res.success ? "Success" : "Failed"}
                        </span>
                      </div>
                      {!res.success && <div className={styles.errorDetails}>{res.error}</div>}
                      {res.success && res.result.message && <div className={styles.successDetails}>{res.result.message}</div>}
                    </div>
                  ))}
                </div>
              )}
            </div>
            <div className={styles.sqlSidebar}>
              <h3>Target Databases</h3>
              <div className={styles.shardList}>
                {productShards.length === 0 ? (
                  <p style={{ fontSize: "12px", color: "var(--text-secondary)" }}>No databases added to this product.</p>
                ) : (
                  productShards.map(s => (
                    <label key={s.name} className={styles.shardSelectItem}>
                      <input 
                        type="checkbox" 
                        checked={targetShards.includes(s.name)} 
                        onChange={() => toggleShard(s.name)}
                      />
                      <div className={styles.shardInfo}>
                        <span className={styles.sName}>{s.name}</span>
                        <span className={styles.sStatus}>{s.status}</span>
                      </div>
                    </label>
                  ))
                )}
              </div>
              <button className="btn btn-glass" style={{ width: "100%", marginTop: "16px" }} onClick={() => setShowProvisionModal(true)}>
                + New Database Shard
              </button>
            </div>
          </div>
        )}

        {activeTab === "shards" && (
          <div className={styles.shardsView}>
            <div className={styles.shardGrid}>
              {productShards.map(s => (
                <div key={s.name} className={styles.shardCard}>
                   <div className={styles.shardHeader}>
                    <span className={styles.shardBadge}>{s.type}</span>
                    <span className={`${styles.statusDot} ${s.status === "Healthy" ? styles.healthy : ""}`}></span>
                  </div>
                  <h3>{s.name}</h3>
                  <div className={styles.tableCounts}>
                    {Object.entries(s.row_counts).map(([t, c]) => (
                      <div key={t} className={styles.tableRow}>
                        <span>{t}</span>
                        <span>{c} rows</span>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
              <div className={styles.provisionCard} onClick={() => setShowProvisionModal(true)}>
                <span>+ Add / Provision New Shard</span>
                <p>Schema will auto-sync upon creation.</p>
              </div>
            </div>
          </div>
        )}

        {activeTab === "tables" && (
          <div className={styles.tablesView}>
             {productShards.length > 0 ? (
               <div className={styles.schemaExplorer}>
                 <p>Aggregate schema view for <strong>{selectedProduct.name}</strong></p>
                 <div className={styles.tableList}>
                   {/* We take tables from the first shard as representative */}
                   {Object.keys(productShards[0].row_counts).map(tableName => (
                     <div key={tableName} className={styles.tableItem}>
                       <div className={styles.tableIcon}>📊</div>
                       <div className={styles.tableName}>{tableName}</div>
                       <div className={styles.tableActions}>
                         <button onClick={() => {
                           setSqlQuery(`SELECT * FROM ${tableName} LIMIT 50;`);
                           setActiveTab("sql");
                         }}>Query</button>
                       </div>
                     </div>
                   ))}
                 </div>
               </div>
             ) : (
               <p>No databases found. Add a shard to see tables.</p>
             )}
          </div>
        )}
      </div>

      {showProvisionModal && (
        <div className={styles.modalOverlay}>
          <div className={styles.modal}>
            <h2>Deploy New Database Shard</h2>
            <p>This shard will be dedicated to <strong>{selectedProduct.name}</strong> and will automatically inherit its master schema.</p>
            <form onSubmit={handleProvisionShard}>
              <input 
                type="text" 
                placeholder="Database Name (e.g. shard-eu-1)" 
                className={styles.input}
                value={newShardName}
                onChange={(e) => setNewShardName(e.target.value)}
                required
              />
              <div className={styles.modalActions}>
                <button type="button" className="btn btn-glass" onClick={() => setShowProvisionModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={provisioning}>
                  {provisioning ? "Provisioning..." : "Provision & Sync"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
