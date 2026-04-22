"use client";

import React, { useState } from "react";
import styles from "../page.module.css";

interface Product {
  name: string;
  slug: string;
  description?: string;
  client_id?: string;
  client_secret?: string;
  access_token?: string;
  gateway_code?: string;
}

interface OverviewTabProps {
  product: Product;
}

export default function OverviewTab({ product }: OverviewTabProps) {
  const [showToken, setShowToken] = useState(false);
  const [showSecret, setShowSecret] = useState(false);
  const [copyStatus, setCopyStatus] = useState<string | null>(null);

  const gatewayUrl = `bdn-bfobs://${product.slug}/${product.gateway_code || "provisioning"}/gateway/`;

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopyStatus(label);
    setTimeout(() => setCopyStatus(null), 2000);
  };

  return (
    <div className={styles.tabContent}>
      <div className={styles.overviewGrid}>
        <div className={`glass-panel ${styles.mainCard}`}>
           <div className={styles.cardHeader}>
              <div className={styles.titleSection}>
                <span className={styles.badge}>INFRASTRUCTURE CORE</span>
                <h2>{product.name}</h2>
              </div>
              <div className={styles.statusGroup}>
                <span className={styles.statusBadge}>HEALTHY</span>
              </div>
           </div>
           
           <p className={styles.description}>{product.description || "Unified cluster management for platform assets."}</p>
           
           <div className={styles.blueprintDivider} />

           <div className={styles.credentials}>
              <h3>Authentication Blueprint</h3>
              
              <div className={styles.credGrid}>
                <div className={styles.field}>
                  <label>Infrastructure ID</label>
                  <div className={styles.valueRow}>
                    <code>{product.client_id || "PROVISIONING..."}</code>
                    <button className={styles.copyBtn} onClick={() => copyToClipboard(product.client_id || "", "id")}>
                      {copyStatus === "id" ? "✓" : "COPY"}
                    </button>
                  </div>
                </div>

                <div className={styles.field}>
                  <label>Security Secret</label>
                  <div className={styles.valueRow}>
                    <code>{showSecret ? (product.client_secret || "N/A") : "••••••••••••••••••••••••"}</code>
                    <button className={styles.copyBtn} onClick={() => setShowSecret(!showSecret)}>
                      {showSecret ? "HIDE" : "SHOW"}
                    </button>
                    {showSecret && (
                      <button className={styles.copyBtn} onClick={() => copyToClipboard(product.client_secret || "", "secret")}>
                        {copyStatus === "secret" ? "✓" : "COPY"}
                      </button>
                    )}
                  </div>
                </div>

                <div className={styles.field}>
                  <label>Product Access Token <span className={styles.hint}>(API Key)</span></label>
                  <div className={styles.valueRow}>
                    <code style={{color:'var(--primary)'}}>{showToken ? (product.access_token || "N/A") : "bfobs_••••••••••••••••••••"}</code>
                    <button className={styles.copyBtn} onClick={() => setShowToken(!showToken)}>
                      {showToken ? "HIDE" : "SHOW"}
                    </button>
                    {showToken && (
                      <button className={styles.copyBtn} onClick={() => copyToClipboard(product.access_token || "", "token")}>
                        {copyStatus === "token" ? "✓" : "COPY"}
                      </button>
                    )}
                  </div>
                </div>
              </div>
           </div>
        </div>

        <div className={styles.sidebar}>
           <div className={`glass-panel ${styles.miniCard}`}>
              <h4>System Pulse</h4>
              <ul className={styles.pulseList}>
                <li><span>Database Cluster</span> <span className={styles.pulseOk}>ACTIVE</span></li>
                <li><span>Object Storage</span> <span className={styles.pulseOk}>LINKED</span></li>
                <li><span>Traffic Gateway</span> <span className={styles.pulseOk}>STABLE</span></li>
              </ul>
           </div>

           <div className={`glass-panel ${styles.miniCard}`}>
              <h4>Endpoint Protocols</h4>
              <div className={styles.protocolStack}>
                 <div className={styles.protocolItem}>
                    <div className={styles.protoHeader}>
                      <span className={styles.protoLabel}>GATEWAY PROTOCOL</span>
                      <button className={styles.miniCopy} onClick={() => copyToClipboard(gatewayUrl, "gate")}>
                        {copyStatus === "gate" ? "✓" : "COPY"}
                      </button>
                    </div>
                    <code className={styles.protoValue}>{gatewayUrl}</code>
                 </div>

                 <div className={styles.protocolItem}>
                    <div className={styles.protoHeader}>
                      <span className={styles.protoLabel}>DATABASE PROXY</span>
                      <button className={styles.miniCopy} onClick={() => copyToClipboard(`/db/p/${product.slug}/execute`, "db")}>
                        {copyStatus === "db" ? "✓" : "COPY"}
                      </button>
                    </div>
                    <code className={styles.protoValue}>/db/p/{product.slug}/execute</code>
                 </div>

                 <div className={styles.protocolItem}>
                    <div className={styles.protoHeader}>
                      <span className={styles.protoLabel}>STORAGE CDN</span>
                      <button className={styles.miniCopy} onClick={() => copyToClipboard(`/storage/view/${product.slug}/{bucket}/{file}`, "cdn")}>
                        {copyStatus === "cdn" ? "✓" : "COPY"}
                      </button>
                    </div>
                    <code className={styles.protoValue}>/storage/view/{product.slug}/...</code>
                 </div>
              </div>
           </div>

           <div className={`glass-panel ${styles.miniCard}`}>
              <h4>Access Guidelines</h4>
              <ul className={styles.guidelineList}>
                <li>Never expose access_token on client-side</li>
                <li>Use .sql migrations for schema changes</li>
                <li>Verify HMAC signatures for all callbacks</li>
              </ul>
           </div>
        </div>
      </div>
  );
}
