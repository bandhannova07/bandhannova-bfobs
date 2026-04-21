"use client";

import React from "react";
import styles from "../page.module.css";

interface Product {
  name: string;
  slug: string;
  description?: string;
  client_id?: string;
  client_secret?: string;
}

interface OverviewTabProps {
  product: Product;
}

export default function OverviewTab({ product }: OverviewTabProps) {
  return (
    <div className={styles.tabContent}>
      <div className={styles.overviewGrid}>
        <div className={`glass-panel ${styles.mainCard}`}>
           <h2>Core System: {product.name}</h2>
           <p className={styles.description}>{product.description || "No metadata provided for this cluster."}</p>
           
           <div className={styles.credentials}>
              <h3>Gateway Credentials</h3>
              <div className={styles.field}>
                <label>Infrastructure ID</label>
                <code>{product.client_id || "PROVISIONING..."}</code>
              </div>
              <div className={styles.field}>
                <label>Security Secret</label>
                <code>{product.client_secret || "••••••••••••••••"}</code>
              </div>
           </div>
        </div>

        <div className={styles.sidebarCards}>
           <div className={`glass-panel ${styles.miniCard}`}>
              <h4>Fleet Health</h4>
              <ul className={styles.statusList}>
                <li><span>Database Cluster</span> <span className={styles.statusOk}>ONLINE</span></li>
                <li><span>Storage LFS</span> <span className={styles.statusOk}>ACTIVE</span></li>
                <li><span>Global Gateway</span> <span className={styles.statusOk}>OPTIMIZED</span></li>
              </ul>
           </div>

           <div className={`glass-panel ${styles.miniCard}`}>
              <h4>Network Protocols</h4>
              <div className={styles.protocolList}>
                 <div className={styles.protocol}>
                    <span>Root Gate</span>
                    <code>bdn-infra://{product.slug}/gate</code>
                 </div>
                 <div className={styles.protocol}>
                    <span>SQL Entry</span>
                    <code>bdn-infra://{product.slug}/sql</code>
                 </div>
                 <div className={styles.protocol}>
                    <span>CDN Hook</span>
                    <code>bdn-infra://{product.slug}/cdn</code>
                 </div>
              </div>
           </div>
        </div>
      </div>
    </div>
  );
}

