"use client";
import React, { useState, useEffect } from "react";
import styles from "../page.module.css";
import { fetchAPI } from "../../../../../lib/api";

interface DatabaseViewerProps {
  shard: {
    id: string;
    name: string;
    db_url: string;
  };
  onClose: () => void;
}

export default function DatabaseViewer({ shard, onClose }: DatabaseViewerProps) {
  const [tables, setTables] = useState<string[]>([]);
  const [selectedTable, setSelectedTable] = useState<string | null>(null);
  const [columns, setColumns] = useState<any[]>([]);
  const [rows, setRows] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [searching, setSearching] = useState(false);

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
    setSearching(true);
    try {
      // Fetch Schema
      const schemaRes = await fetchAPI(`/admin/infrastructure/shards/${shard.id}/query`, {
        method: "POST",
        body: JSON.stringify({ query: `PRAGMA table_info(${tableName})` })
      });
      
      // Fetch Data
      const dataRes = await fetchAPI(`/admin/infrastructure/shards/${shard.id}/query`, {
        method: "POST",
        body: JSON.stringify({ query: `SELECT * FROM ${tableName} LIMIT 100` })
      });

      if (schemaRes.success) setColumns(schemaRes.data);
      if (dataRes.success) setRows(dataRes.data);
    } catch (err) {
      console.error(err);
    } finally {
      setSearching(false);
    }
  };

  return (
    <div className={styles.viewerOverlay}>
      <div className={`glass-panel ${styles.viewerContainer}`}>
        {/* ─── Viewer Header ─────────────────────────── */}
        <div className={styles.viewerHeader}>
           <div className={styles.viewerTitleArea}>
              <div className={styles.dbIcon}>🗄️</div>
              <div>
                 <h3>{shard.name}</h3>
                 <code className={styles.dbBadge}>{shard.db_url}</code>
              </div>
           </div>
           <button className={styles.closeViewer} onClick={onClose}>✕</button>
        </div>

        <div className={styles.viewerMain}>
           {/* ─── Sidebar: Tables ────────────────────── */}
           <aside className={styles.viewerSidebar}>
              <div className={styles.sidebarHeader}>
                 <span>TABLES</span>
                 <span className={styles.countBadge}>{tables.length}</span>
              </div>
              <div className={styles.tableNav}>
                 {tables.map(table => (
                    <button 
                      key={table} 
                      className={`${styles.tableNavItem} ${selectedTable === table ? styles.activeTable : ""}`}
                      onClick={() => handleTableSelect(table)}
                    >
                       <span className={styles.tIcon}>📊</span>
                       {table}
                    </button>
                 ))}
              </div>
           </aside>

           {/* ─── Content: Data Grid ─────────────────── */}
           <main className={styles.viewerContent}>
              {searching ? (
                <div className={styles.gridLoading}>
                   <div className={styles.spinner}></div>
                   <p>FETCHING DATA FROM SHARD...</p>
                </div>
              ) : selectedTable ? (
                <div className={styles.gridWrapper}>
                   <div className={styles.gridHeader}>
                      <div className={styles.gridMeta}>
                         <h4>{selectedTable}</h4>
                         <p>{rows.length} rows loaded (Limit 100)</p>
                      </div>
                      <div className={styles.gridActions}>
                         <button className="btn btn-glass" style={{fontSize: '11px'}} onClick={() => handleTableSelect(selectedTable)}>Refresh</button>
                         <button className="btn btn-primary" style={{fontSize: '11px'}}>+ Add Row</button>
                      </div>
                   </div>
                   
                   <div className={styles.tableScroll}>
                      <table className={styles.dataTable}>
                         <thead>
                            <tr>
                               {columns.map(col => (
                                  <th key={col.name}>
                                     <div className={styles.thContent}>
                                        <span className={styles.colName}>{col.name}</span>
                                        <span className={styles.colType}>{col.type}</span>
                                     </div>
                                  </th>
                               ))}
                            </tr>
                         </thead>
                         <tbody>
                            {rows.map((row, idx) => (
                               <tr key={idx}>
                                  {columns.map(col => (
                                     <td key={col.name}>
                                        {row[col.name] === null ? <em className={styles.nullText}>NULL</em> : String(row[col.name])}
                                     </td>
                                  ))}
                               </tr>
                            ))}
                         </tbody>
                      </table>
                   </div>
                </div>
              ) : (
                <div className={styles.noTableSelected}>
                   <div className={styles.bigIcon}>📁</div>
                   <p>Select a table to explore data</p>
                </div>
              )}
           </main>
        </div>
      </div>
    </div>
  );
}
