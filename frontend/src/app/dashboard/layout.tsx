"use client";

import React from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import styles from "./layout.module.css";
import { clearToken } from "../../lib/api";

interface NavItem {
  name: string;
  path: string;
  icon: string;
}

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();

  const handleLogout = () => {
    clearToken();
    router.push("/");
  };

  const navItems: NavItem[] = [
    { name: "Overview", path: "/dashboard", icon: "📊" },
    { name: "API Keys", path: "/dashboard/keys", icon: "🔑" },
    { name: "Products", path: "/dashboard/products", icon: "📦" },
    { name: "Security", path: "/dashboard/security", icon: "🛡️" },
    { name: "Audit Log", path: "/dashboard/audit", icon: "📋" },
  ];

  let currentTitle = "Command Center";
  navItems.forEach(item => {
    if (item.path === "/dashboard") {
       if (pathname === "/dashboard") currentTitle = item.name;
    } else if (pathname.startsWith(item.path)) {
       currentTitle = item.name;
    }
  });

  return (
    <div className={styles.dashboardContainer}>
      {/* Sidebar */}
      <aside className={styles.sidebar}>
        <div className={styles.sidebarHeader}>
          BandhanNova <span>BFOBS</span>
        </div>
        <nav className={styles.nav}>
          {navItems.map((item) => {
            const isActive = item.path === "/dashboard" 
              ? pathname === "/dashboard" 
              : pathname.startsWith(item.path);
              
            return (
              <Link 
                key={item.path} 
                href={item.path}
                className={`${styles.navItem} ${isActive ? styles.navItemActive : ""}`}
              >
                <span>{item.icon}</span>
                {item.name}
              </Link>
            );
          })}
        </nav>
        <button onClick={handleLogout} className={styles.logoutBtn}>
          LOGOUT SYSTEM
        </button>
      </aside>

      {/* Main Content Area */}
      <div className={styles.mainWrapper}>
        <header className={styles.topbar}>
          <h1 className={styles.pageTitle}>{currentTitle}</h1>
          <div className={styles.topbarActions}>
            <div className={styles.systemStatus}>
              <div className={styles.statusDot}></div>
              <span className={styles.statusText}>SYSTEM SECURE</span>
            </div>
            <button onClick={handleLogout} className={styles.mobileLogoutBtn} title="Logout">
              ⏻
            </button>
          </div>
        </header>
        
        <main className={styles.content}>
          {children}
        </main>
      </div>
    </div>
  );
}

