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
  const [product, setProduct] = useState<any>(null);
  const [activeTab, setActiveTab] = useState("overview");
  const [loading, setLoading] = useState(true);
  const [userRole, setUserRole] = useState<string | null>(null);

  useEffect(() => {
    const role = sessionStorage.getItem("user_role");
    const allowedSlug = sessionStorage.getItem("product_slug");
    setUserRole(role);

    // Security: Developers can only access their assigned product
    if (role === "developer" && slug !== allowedSlug) {
      router.push(`/dashboard/products/${allowedSlug}`);
      return;
    }

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

  const handleLogout = () => {
    sessionStorage.clear();
    router.push(userRole === "developer" ? "/developer/login" : "/");
  };

  if (loading) return <div className={styles.loading}>ESTABLISHING CONNECTION...</div>;
  if (!product) return <div className={styles.error}>INFRASTRUCTURE NOT FOUND</div>;

  return (
    <div className={styles.container}>
      {/* ─── Product Header ─────────────────────────── */}
      <div className={styles.header}>
        <div className={styles.headerLeft}>
          <div className={styles.breadcrumbs}>
            <span className={styles.current}>{product.name}</span>
          </div>
          <div className={styles.titleArea}>
              <div className={styles.icon}>
                 {product.slug === "auth" ? "🔐" : 
                  product.slug === "analytics" ? "📈" : 
                  product.slug === "market" ? "💰" : 
                  product.slug === "ai" ? "🤖" : "📦"}
              </div>
              <div className={styles.titleInfo}>
                 <div style={{display:'flex', alignItems:'center', gap:'10px'}}>
                   <h1 className={styles.title}>{product.name}</h1>
                   <div className={styles.liveBadge}>
                      <span className={styles.pulse}></span>
                      LIVE
                   </div>
                 </div>
                 <code className={styles.url}>bdn-bfobs://{product.slug}/{product.gateway_code || "..."}/gateway</code>
              </div>
          </div>
        </div>

        <div className={styles.headerActions}>
           <button className="btn btn-glass" onClick={() => window.open("/docs", "_blank")}>📖 DOCS</button>
           {userRole === "developer" ? (
             <button className="btn btn-primary" onClick={handleLogout} style={{background:'var(--danger)'}}>EXIT PORTAL</button>
           ) : (
             <button className="btn btn-primary">SETTINGS</button>
           )}
        </div>
      </div>

      {/* ─── Section Navigation ──────────────────────── */}
      <nav className={styles.tabs}>
        <button 
          className={`${styles.tab} ${activeTab === "overview" ? styles.activeTab : ""}`}
          onClick={() => setActiveTab("overview")}
        >
          <span className={styles.tabIcon}>📊</span> Overview
        </button>
        <button 
          className={`${styles.tab} ${activeTab === "database" ? styles.activeTab : ""}`}
          onClick={() => setActiveTab("database")}
        >
          <span className={styles.tabIcon}>🗄️</span> Databases
        </button>
        <button 
          className={`${styles.tab} ${activeTab === "storage" ? styles.activeTab : ""}`}
          onClick={() => setActiveTab("storage")}
        >
          <span className={styles.tabIcon}>☁️</span> Storage
        </button>
        <button 
          className={`${styles.tab} ${activeTab === "sql" ? styles.activeTab : ""}`}
          onClick={() => setActiveTab("sql")}
        >
          <span className={styles.tabIcon}>⚡</span> SQL Forge
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

