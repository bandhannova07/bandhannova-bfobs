"use client";

import React, { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import styles from "./page.module.css";
import { fetchAPI } from "../../../../lib/api";

// Sub-components
import DatabaseView from "./components/DatabaseView";
import StorageView from "./components/StorageView";
import SQLEditor from "./components/SQLEditor";
import OverviewTab from "./components/OverviewTab";

interface Product {
  id: string;
  name: string;
  slug: string;
}

export default function ProductDetailDashboard() {
  const params = useParams();
  const slug = params.slug as string;
  const router = useRouter();
  const [product, setProduct] = useState<Product | null>(null);
  const [activeTab, setActiveTab] = useState("overview");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadProductDetails();
  }, [slug]);

  const loadProductDetails = async () => {
    setLoading(true);
    try {
      const res = await fetchAPI(`/admin/products/${slug}`);
      if (res.success) setProduct(res.product);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <div className={styles.loading}>ESTABLISHING CONNECTION...</div>;
  if (!product) return <div className={styles.error}>INFRASTRUCTURE NOT FOUND</div>;

  return (
    <div className={styles.container}>
      {/* ─── Product Header ─────────────────────────── */}
      <div className={styles.header}>
        <div className={styles.titleArea}>
           <button className={styles.backBtn} onClick={() => router.push("/dashboard/products")}>←</button>
           <div className={styles.icon}>
              {product.slug === "auth" ? "🔐" : 
               product.slug === "analytics" ? "📈" : 
               product.slug === "market" ? "💰" : 
               product.slug === "ai" ? "🤖" : "📦"}
           </div>
           <div>
              <h1 className={styles.title}>{product.name}</h1>
              <code className={styles.url}>bdn-bfobs://{product.slug}/{product.gateway_code || "..."}/gateway/</code>
           </div>
        </div>
        <div className={styles.statusBadge}>
           <div className={styles.dot}></div>
           INFRASTRUCTURE LIVE
        </div>
      </div>

      {/* ─── Navigation Tabs ────────────────────────── */}
      <nav className={styles.tabs}>
        <button 
          className={`${styles.tab} ${activeTab === "overview" ? styles.activeTab : ""}`}
          onClick={() => setActiveTab("overview")}
        >
          Overview
        </button>
        <button 
          className={`${styles.tab} ${activeTab === "database" ? styles.activeTab : ""}`}
          onClick={() => setActiveTab("database")}
        >
          Databases
        </button>
        <button 
          className={`${styles.tab} ${activeTab === "storage" ? styles.activeTab : ""}`}
          onClick={() => setActiveTab("storage")}
        >
          Storage
        </button>
        <button 
          className={`${styles.tab} ${activeTab === "sql" ? styles.activeTab : ""}`}
          onClick={() => setActiveTab("sql")}
        >
          SQL Forge
        </button>
      </nav>

      {/* ─── Tab Content ────────────────────────────── */}
      <div className={styles.content}>
        {activeTab === "overview" && <OverviewTab product={product} />}
        {activeTab === "database" && <DatabaseView product={product} />}
        {activeTab === "storage" && <StorageView product={product} />}
        {activeTab === "sql" && <SQLEditor product={product} />}
      </div>
    </div>
  );
}

