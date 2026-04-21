"use client";

import React, { useState, useEffect } from "react";
import styles from "../page.module.css";
import { fetchAPI } from "../../../../../lib/api";

interface Shard {
  id: string;
  name: string;
  db_url: string;
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

  const loadShards = async () => {
    setLoading(true);
    try {
      const res = await fetchAPI(`/admin/databases?product_slug=${product.slug}`);
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

  const handleProvision = async () => {
    if (!confirm("Provision a new dedicated database shard for this product?")) return;
    try {
      const res = await fetchAPI("/admin/db/provision", {
        method: "POST",
        body: JSON.stringify({ product_id: product.id, product_slug: product.slug, name: `${product.slug}-shard-${shards.length + 1}` })
      });
      if (res.success) {
        alert("Success! Database shard provisioned and master schema synced.");
        loadShards();
      }
    } catch (err: any) {
      alert("Error: " + err.message);
    }
  };

  const handleRemove = async (id: string) => {
    if (!confirm("Are you sure you want to PERMANENTLY remove this database shard? This cannot be undone.")) return;
    try {
      const res = await fetchAPI(`/admin/db/remove/${id}`, { method: "POST" });
      if (res.success) {
        alert("Shard decommissioned successfully.");
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
        <button className="btn btn-primary" onClick={handleProvision}>+ Provision New Shard</button>
      </div>

      {loading ? (
        <div className={styles.loading}>SYNCING FLEET...</div>
      ) : shards.length === 0 ? (
        <div className={styles.emptyState}>
           <p>No databases provisioned for this product yet.</p>
           <button className="btn btn-glass" onClick={handleProvision}>Deploy First Shard</button>
        </div>
      ) : (
        <div className={styles.dbList}>
          {shards.map(db => (
            <div key={db.id} className={`glass-panel ${styles.dbCard}`}>
               <div className={styles.dbCardLeft}>
                 <div className={styles.dbIcon}>🗄️</div>
                 <div className={styles.dbInfo}>
                    <h4>{db.name}</h4>
                    <code>{db.db_url}</code>
                 </div>
               </div>
               <div className={styles.dbCardRight}>
                  <div className={styles.badgeOnline}>ACTIVE</div>
                  <button className={styles.removeBtn} onClick={() => handleRemove(db.id)}>Decommission</button>
               </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

