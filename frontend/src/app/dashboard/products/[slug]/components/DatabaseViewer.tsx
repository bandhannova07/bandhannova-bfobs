"use client";
import React, { useState, useEffect } from "react";
import styles from "../page.module.css";
import { fetchAPI } from "../../../../../lib/api";

interface ShardStudioProps {
  shard: {
    id: string;
    name: string;
    db_url: string;
  };
  onClose: () => void;
}

export default function DatabaseViewer({ shard, onClose }: ShardStudioProps) {
  const [tables, setTables] = useState<string[]>([]);
  const [selectedTable, setSelectedTable] = useState<string | null>(null);
  const [columns, setColumns] = useState<any[]>([]);
  const [rows, setRows] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(false);

  useEffect(() => {
    loadTables();
  }, [shard.id]);

  const loadTables = async () => {
    setLoading(true);
    try {
      const res = await fetchAPI(`/admin/infrastructure/shards/${shard.id}/query`, {
        method: "POST",
        body: JSON.stringify({ query: "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT LIKE '_%'" })
      });
      if (res.success) {
        const tableNames = res.data.map((t: any) => t.name);
        setTables(tableNames);
        if (tableNames.length > 0) handleTableSelect(tableNames[0]);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleTableSelect = async (tableName: string) => {
    setSelectedTable(tableName);
    setFetching(true);
    try {
      // Fetch Schema using PRAGMA
      const schemaRes = await fetchAPI(`/admin/infrastructure/shards/${shard.id}/query`, {
        method: "POST",
        body: JSON.stringify({ query: `PRAGMA table_info("${tableName}")` })
      });
      
      // Fetch Data
      const dataRes = await fetchAPI(`/admin/infrastructure/shards/${shard.id}/query`, {
        method: "POST",
        body: JSON.stringify({ query: `SELECT * FROM "${tableName}" LIMIT 200` })
      });

      if (schemaRes.success) setColumns(schemaRes.data);
      if (dataRes.success) setRows(dataRes.data);
    } catch (err) {
      console.error(err);
    } finally {
      setFetching(false);
    }
  };

  return (
    <div className={styles.studioRoot}>
      {/* ─── Top Bar: Orchestration Control ────────── */}
      <header className={styles.studioTopBar}>
         <div className={styles.studioBrand}>
            <div className={styles.studioIcon}>⚡</div>
            <div className={styles.studioBreadcrumbs}>
               <span className={styles.crumbProject}>Project</span>
               <span className={styles.crumbDivider}>/</span>
               <span className={styles.crumbShard}>{shard.name}</span>
               {selectedTable && (
                 <>
                   <span className={styles.crumbDivider}>/</span>
                   <span className={styles.crumbTable}>{selectedTable}</span>
                 </>
               )}
            </div>
         </div>
         <div className={styles.studioTopActions}>
            <div className={styles.connectionStatus}>
               <div className={styles.livePulse}></div>
               <span>SHARD CONNECTED</span>
            </div>
            <button className={styles.exitStudio} onClick={onClose}>
               <span>Exit Studio</span>
               <kbd>ESC</kbd>
            </button>
         </div>
      </header>

      <div className={styles.studioLayout}>
         {/* ─── Sidebar: Table Explorer ──────────────── */}
         <aside className={styles.studioSidebar}>
            <div className={styles.sidebarSection}>
               <div className={styles.sidebarLabel}>DATABASE</div>
               <div className={styles.sidebarNav}>
                  <button className={`${styles.navItem} ${styles.navActive}`}>
                     <span className={styles.navIcon}>📁</span>
                     Tables
                  </button>
                  <button className={styles.navItem}>
                     <span className={styles.navIcon}>🔍</span>
                     SQL Editor
                  </button>
                  <button className={styles.navItem}>
                     <span className={styles.navIcon}>🛡️</span>
                     Policies (RLS)
                  </button>
               </div>
            </div>

            <div className={styles.sidebarSection}>
               <div className={styles.sidebarLabel}>ALL TABLES</div>
               <div className={styles.tableListScroll}>
                  {loading ? (
                    <div className={styles.sidebarLoading}>Loading schema...</div>
                  ) : tables.map(table => (
                    <button 
                      key={table} 
                      className={`${styles.tableBtn} ${selectedTable === table ? styles.tableActive : ""}`}
                      onClick={() => handleTableSelect(table)}
                    >
                       <span className={styles.tableIconMini}>#</span>
                       {table}
                    </button>
                  ))}
               </div>
            </div>
         </aside>

         {/* ─── Main Content: Data Studio ────────────── */}
         <main className={styles.studioMain}>
            <div className={styles.studioToolbox}>
               <div className={styles.toolboxLeft}>
                  <button className={styles.toolBtn}>
                    <span>Filter</span>
                  </button>
                  <button className={styles.toolBtn}>
                    <span>Sort</span>
                  </button>
               </div>
               <div className={styles.toolboxRight}>
                  <button className={styles.toolBtnPrimary}>+ Insert Row</button>
                  <button className={styles.toolBtn}>Export</button>
               </div>
            </div>

            <div className={styles.gridContainer}>
               {fetching ? (
                 <div className={styles.gridOverlay}>
                    <div className={styles.studioSpinner}></div>
                    <p>Synchronizing data shards...</p>
                 </div>
               ) : selectedTable ? (
                 <div className={styles.spreadsheet}>
                    <table className={styles.studioTable}>
                       <thead>
                          <tr>
                             <th className={styles.rowNumberCol}>#</th>
                             {columns.map(col => (
                                <th key={col.name}>
                                   <div className={styles.thContent}>
                                      <span className={styles.thType}>
                                         {col.type.includes("INT") ? "123" : "abc"}
                                      </span>
                                      <span className={styles.thName}>{col.name}</span>
                                      {col.pk === 1 && <span className={styles.pkBadge}>PK</span>}
                                   </div>
                                </th>
                             ))}
                          </tr>
                       </thead>
                       <tbody>
                          {rows.map((row, idx) => (
                             <tr key={idx}>
                                <td className={styles.rowNumber}>{idx + 1}</td>
                                {columns.map(col => (
                                   <td key={col.name}>
                                      <div className={styles.cellContent}>
                                         {row[col.name] === null ? (
                                           <span className={styles.nullCell}>NULL</span>
                                         ) : String(row[col.name])}
                                      </div>
                                   </td>
                                ))}
                             </tr>
                          ))}
                       </tbody>
                    </table>
                 </div>
               ) : (
                 <div className={styles.noTableState}>
                    <div className={styles.bigStudioIcon}>📊</div>
                    <h2>No table selected</h2>
                    <p>Select a table from the sidebar to begin orchestration.</p>
                 </div>
               )}
            </div>

            <footer className={styles.studioFooter}>
               <div className={styles.footerInfo}>
                  {selectedTable && (
                    <>
                      <span>Rows: {rows.length}</span>
                      <span className={styles.footerSep}>|</span>
                      <span>Columns: {columns.length}</span>
                    </>
                  )}
               </div>
               <div className={styles.footerBrand}>
                  BandhanNova Shard Studio v1.0
               </div>
            </footer>
         </main>
      </div>
    </div>
  );
}
