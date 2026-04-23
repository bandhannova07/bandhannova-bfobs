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

  const gatewayUrl = `bdn-bfobs://${product.slug}/${product.gateway_code || "provisioning"}/gateway`;

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopyStatus(label);
    setTimeout(() => setCopyStatus(null), 2000);
  };

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
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                <code>{product.client_id || "PROVISIONING..."}</code>
                <button className="btn btn-glass" style={{ fontSize: '10px', padding: '4px 8px' }} onClick={() => copyToClipboard(product.client_id || "", "id")}>
                  {copyStatus === "id" ? "✓" : "Copy"}
                </button>
              </div>
            </div>

            <div className={styles.field}>
              <label>Security Secret</label>
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                <code>{showSecret ? (product.client_secret || "N/A") : "••••••••••••••••"}</code>
                <button className="btn btn-glass" style={{ fontSize: '10px', padding: '4px 8px' }} onClick={() => setShowSecret(!showSecret)}>
                  {showSecret ? "Hide" : "Show"}
                </button>
                {showSecret && (
                  <button className="btn btn-glass" style={{ fontSize: '10px', padding: '4px 8px' }} onClick={() => copyToClipboard(product.client_secret || "", "secret")}>
                    {copyStatus === "secret" ? "✓" : "Copy"}
                  </button>
                )}
              </div>
            </div>

            <div className={styles.field}>
              <label>Product Access Token <span style={{ fontSize: '10px', color: '#10b981' }}>(API Key for Developers)</span></label>
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                <code style={{ color: '#10b981' }}>{showToken ? (product.access_token || "N/A") : "bfobs_••••••••••••••••"}</code>
                <button className="btn btn-glass" style={{ fontSize: '10px', padding: '4px 8px' }} onClick={() => setShowToken(!showToken)}>
                  {showToken ? "Hide" : "Show"}
                </button>
                {showToken && (
                  <button className="btn btn-glass" style={{ fontSize: '10px', padding: '4px 8px' }} onClick={() => copyToClipboard(product.access_token || "", "token")}>
                    {copyStatus === "token" ? "✓" : "Copy"}
                  </button>
                )}
              </div>
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
            <h4>Access Protocols</h4>
            <div className={styles.protocolList}>
              <div className={styles.protocol}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <span>Gateway URL</span>
                  <button className="btn btn-glass" style={{ fontSize: '9px', padding: '2px 6px' }} onClick={() => copyToClipboard(gatewayUrl, "gate")}>
                    {copyStatus === "gate" ? "✓" : "Copy"}
                  </button>
                </div>
                <code style={{ color: '#10b981' }}>{gatewayUrl}</code>
              </div>

              <div className={styles.protocol}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <span>Database Proxy</span>
                  <button className="btn btn-glass" style={{ fontSize: '9px', padding: '2px 6px' }} onClick={() => copyToClipboard(`/db/p/${product.slug}/execute`, "db")}>
                    {copyStatus === "db" ? "✓" : "Copy"}
                  </button>
                </div>
                <code>/db/p/{product.slug}/execute</code>
              </div>

              <div className={styles.protocol}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <span>Storage CDN</span>
                  <button className="btn btn-glass" style={{ fontSize: '9px', padding: '2px 6px' }} onClick={() => copyToClipboard(`/storage/view/${product.slug}/{bucket}/{file}`, "cdn")}>
                    {copyStatus === "cdn" ? "✓" : "Copy"}
                  </button>
                </div>
                <code>/storage/view/{product.slug}/&#123;bucketName&#125;/&#123;filePath&#125;</code>
              </div>
            </div>
          </div>

          <div className={`glass-panel ${styles.miniCard}`}>
            <h4>Developer Rules</h4>
            <ul className={styles.statusList}>
              <li style={{ fontSize: '11px', color: '#aaa' }}>⚠️ Never expose access_token in client-side code</li>
              <li style={{ fontSize: '11px', color: '#aaa' }}>📁 Use .sql files for database migrations</li>
              <li style={{ fontSize: '11px', color: '#aaa' }}>🔐 Auth via BandhanNova default auth shards</li>
              <li style={{ fontSize: '11px', color: '#aaa' }}>🔑 Store all secrets in .env files only</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}
