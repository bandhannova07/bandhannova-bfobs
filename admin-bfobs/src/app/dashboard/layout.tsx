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
  const [userRole, setUserRole] = React.useState<string | null>(null);

  React.useEffect(() => {
    setUserRole(sessionStorage.getItem("user_role"));
  }, []);

  const handleLogout = () => {
    clearToken();
    const role = sessionStorage.getItem("user_role");
    sessionStorage.clear();
    router.push(role === "developer" ? "/developer/login" : "/");
  };

  const navItems: NavItem[] = [
    { name: "Overview", path: "/dashboard", icon: "📊" },
    { name: "API Keys", path: "/dashboard/keys", icon: "🔑" },
    { name: "Products", path: "/dashboard/products", icon: "📦" },
    { name: "Documentation", path: "/dashboard/docs", icon: "📚" },
    { name: "Default Shards", path: "/dashboard/infrastructure", icon: "⚙️" },
    { name: "Security", path: "/dashboard/security", icon: "🛡️" },
    { name: "Audit Log", path: "/dashboard/audit", icon: "📋" },
  ];

  const isDeveloper = userRole === "developer";
  let currentTitle = isDeveloper ? "Product Portal" : "Command Center";
  
  if (!isDeveloper) {
    navItems.forEach(item => {
      if (item.path === "/dashboard") {
         if (pathname === "/dashboard") currentTitle = item.name;
      } else if (pathname.startsWith(item.path)) {
         currentTitle = item.name;
      }
    });
  }

  return (
    <div className={styles.dashboardContainer}>
      {/* Sidebar - Hidden for Developers */}
      {!isDeveloper && (
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
      )}

      {/* Main Content Area - Full width for Developers */}
      <div className={styles.mainWrapper} style={isDeveloper ? { marginLeft: 0, width: '100%' } : {}}>
        <header className={styles.topbar}>
          <h1 className={styles.pageTitle}>{currentTitle}</h1>
          <div className={styles.topbarActions}>
            <div className={styles.systemStatus}>
              <div className={styles.statusDot} style={isDeveloper ? { background: '#10b981' } : {}}></div>
              <span className={styles.statusText}>{isDeveloper ? "PORTAL ACTIVE" : "SYSTEM SECURE"}</span>
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

