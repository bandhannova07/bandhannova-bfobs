"use client";
import React, { useState, useEffect, useMemo } from "react";
import styles from "../page.module.css";
import { fetchAPI } from "../../../../../lib/api";

interface ShardStudioProps {
  shard: {
    id: string;
    name: string;
    type: string; // Shard Category/Type
  };
  productSlug?: string; // Optional for product-specific shards
  onClose: () => void;
}

export default function DatabaseViewer({ shard, productSlug, onClose }: ShardStudioProps) {
  const [tables, setTables] = useState<string[]>([]);
  const [selectedTable, setSelectedTable] = useState<string | null>(null);
  const [columns, setColumns] = useState<any[]>([]);
  const [rows, setRows] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Studio UI States
  const [searchTerm, setSearchTerm] = useState("");
  const [sortConfig, setSortConfig] = useState<{ key: string; direction: "asc" | "desc" | null }>({ key: "", direction: null });
  const [isSelectMode, setIsSelectMode] = useState(false);
  const [selectedRowIds, setSelectedRowIds] = useState<Set<any>>(new Set());
  const [editingCell, setEditingCell] = useState<{ rowIdx: number; colName: string; value: any } | null>(null);

  useEffect(() => {
    loadTables();
  }, [shard.id]);

  const execSQL = async (sql: string) => {
    // Admin uses the specific shard query endpoint
    return await fetchAPI(`/admin/infrastructure/shards/${shard.id}/query`, {
      method: "POST",
      body: JSON.stringify({ query: sql }),
    });
  };

  const loadTables = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await execSQL("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'");
      if (res.success && res.data) {
        const tableNames = res.data.map((t: any) => t.name || Object.values(t)[0]);
        setTables(tableNames);
        if (tableNames.length > 0 && !selectedTable) {
          handleTableSelect(tableNames[0]);
        }
      } else {
        setError("No tables found in this shard cluster.");
      }
    } catch (err: any) {
      setError(err.message || "Connection to fleet failed");
    } finally {
      setLoading(false);
    }
  };

  const handleTableSelect = async (tableName: string) => {
    setSelectedTable(tableName);
    setFetching(true);
    setError(null);
    setSearchTerm("");
    setSelectedRowIds(new Set());
    setIsSelectMode(false);
    try {
      const schemaRes = await execSQL(`PRAGMA table_info("${tableName}")`);
      const dataRes = await execSQL(`SELECT * FROM "${tableName}" LIMIT 1000`);

      if (schemaRes.success) setColumns(schemaRes.data || []);
      if (dataRes.success) setRows(dataRes.data || []);
    } catch (err: any) {
      setError(err.message || "Data retrieval failed");
    } finally {
      setFetching(false);
    }
  };

  // ─── Data Orchestration Logic ───────────────────
  const filteredRows = useMemo(() => {
    let result = [...rows];
    if (searchTerm) {
      result = result.filter(row => 
        Object.values(row).some(val => String(val).toLowerCase().includes(searchTerm.toLowerCase()))
      );
    }
    if (sortConfig.key && sortConfig.direction) {
      result.sort((a, b) => {
        const valA = a[sortConfig.key];
        const valB = b[sortConfig.key];
        if (valA < valB) return sortConfig.direction === "asc" ? -1 : 1;
        if (valA > valB) return sortConfig.direction === "asc" ? 1 : -1;
        return 0;
      });
    }
    return result;
  }, [rows, searchTerm, sortConfig]);

  const handleSort = (key: string) => {
    setSortConfig(prev => ({
      key,
      direction: prev.key === key && prev.direction === "asc" ? "desc" : "asc"
    }));
  };

  // ─── Fleet Synchronization Actions ────────────────
  const execCategorySQL = async (sql: string) => {
    // Admin has a special bulk category endpoint to sync changes across all shards of the same type
    return await fetchAPI(`/admin/db/execute-category`, {
      method: "POST",
      body: JSON.stringify({ category: shard.type, sql }),
    });
  };

  const dropTable = async (tableName: string) => {
    if (!confirm(`⚠️ FLEET ACTION: Drop table "${tableName}" across ALL "${shard.type}" shards in the registry?`)) return;
    setFetching(true);
    try {
      const res = await execCategorySQL(`DROP TABLE "${tableName}"`);
      if (res.success) { setSelectedTable(null); loadTables(); }
      else alert("Fleet operation failed: " + res.message);
    } catch (err: any) { alert("Error: " + err.message); } finally { setFetching(false); }
  };

  const renameTable = async (oldName: string) => {
    const newName = prompt("New name for table:", oldName);
    if (!newName || newName === oldName) return;
    if (!confirm(`⚠️ FLEET ACTION: Rename to "${newName}" across the entire "${shard.type}" fleet?`)) return;
    setFetching(true);
    try {
      const res = await execCategorySQL(`ALTER TABLE "${oldName}" RENAME TO "${newName}"`);
      if (res.success) { setSelectedTable(newName); loadTables(); }
      else alert(res.message);
    } catch (err: any) { alert("Error: " + err.message); } finally { setFetching(false); }
  };

  const dropColumn = async (colName: string) => {
    if (!selectedTable) return;
    if (!confirm(`⚠️ FLEET ACTION: Drop column "${colName}" across the entire "${shard.type}" fleet?`)) return;
    setFetching(true);
    try {
      const res = await execCategorySQL(`ALTER TABLE "${selectedTable}" DROP COLUMN "${colName}"`);
      if (res.success) handleTableSelect(selectedTable);
      else alert(res.message);
    } catch (err: any) { alert("Error: " + err.message); } finally { setFetching(false); }
  };

  const renameColumn = async (oldCol: string) => {
    if (!selectedTable) return;
    const newCol = prompt("New column name:", oldCol);
    if (!newCol || newCol === oldCol) return;
    if (!confirm(`⚠️ FLEET ACTION: Rename column across the entire "${shard.type}" fleet?`)) return;
    setFetching(true);
    try {
      const res = await execCategorySQL(`ALTER TABLE "${selectedTable}" RENAME COLUMN "${oldCol}" TO "${newCol}"`);
      if (res.success) handleTableSelect(selectedTable);
      else alert(res.message);
    } catch (err: any) { alert("Error: " + err.message); } finally { setFetching(false); }
  };

  const saveCellEdit = async (row: any, colName: string, newValue: any) => {
    if (!selectedTable) return;
    const primaryKey = columns.find(c => c.pk === 1)?.name || columns[0]?.name;
    const pkValue = row[primaryKey];
    setFetching(true);
    try {
      const formattedValue = typeof newValue === 'string' ? `'${newValue}'` : newValue;
      const formattedPk = typeof pkValue === 'string' ? `'${pkValue}'` : pkValue;
      const res = await execSQL(`UPDATE "${selectedTable}" SET "${colName}" = ${formattedValue} WHERE "${primaryKey}" = ${formattedPk}`);
      if (res.success) {
        setRows(rows.map(r => r[primaryKey] === pkValue ? { ...r, [colName]: newValue } : r));
        setEditingCell(null);
      } else alert(res.message);
    } catch (err: any) { alert("Error: " + err.message); } finally { setFetching(false); }
  };

  const toggleRowSelection = (id: any) => {
    const next = new Set(selectedRowIds);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setSelectedRowIds(next);
  };

  const deleteSelectedRows = async () => {
    if (selectedRowIds.size === 0 || !selectedTable) return;
    const primaryKey = columns.find(c => c.pk === 1)?.name || columns[0]?.name;
    if (!confirm(`Delete ${selectedRowIds.size} rows in current shard?`)) return;
    setFetching(true);
    try {
      const idsToDelete = Array.from(selectedRowIds).map(id => typeof id === 'string' ? `'${id}'` : id).join(",");
      const res = await execSQL(`DELETE FROM "${selectedTable}" WHERE "${primaryKey}" IN (${idsToDelete})`);
      if (res.success) handleTableSelect(selectedTable);
      else alert(res.message);
    } catch (err: any) { alert("Error: " + err.message); } finally { setFetching(false); }
  };

  const resetFleet = async () => {
    if (!productSlug) return alert("System infrastructure cannot be reset via Fleet Studio. Use Core Master tools.");
    const infraId = prompt("⚠️ CRITICAL ACTION: Enter Infrastructure ID to factory reset ALL shards for this product:");
    if (!infraId) return;

    if (!confirm("ARE YOU ABSOLUTELY SURE? This will DELETE ALL DATA across the entire fleet and cannot be undone!")) return;

    setFetching(true);
    try {
      const res = await fetchAPI(`/admin/db/reset-fleet`, {
        method: "POST",
        body: JSON.stringify({
          product_slug: productSlug,
          infra_id: infraId
        }),
      });

      if (res.success) {
        alert("Fleet successfully reset to factory defaults.");
        setSelectedTable(null);
        loadTables();
      } else {
        alert("Reset failed: " + res.message);
      }
    } catch (err: any) {
      alert("Error: " + err.message);
    } finally {
      setFetching(false);
    }
  };

  return (
    <div className={styles.studioRoot}>
      <header className={styles.studioTopBar}>
         <div className={styles.studioBrand}>
            <div className={styles.studioIcon}>⚡</div>
            <div className={styles.studioBreadcrumbs}>
               <span className={styles.crumbProject}>Fleet Master Studio</span>
               <span className={styles.crumbDivider}>/</span>
               <span className={styles.crumbShard}>{shard.name} ({shard.type})</span>
            </div>
         </div>
         <div className={styles.studioTopActions}>
            <div className={styles.connectionStatus}>
               <div className={styles.livePulse}></div>
               <span>FLEET SYNC ACTIVE</span>
            </div>
            <button className={styles.exitStudioSimple} onClick={onClose}>Exit Studio</button>
         </div>
      </header>

      <div className={styles.studioLayout}>
         <aside className={styles.studioSidebar}>
            <div className={styles.sidebarSection}>
               <div className={styles.sidebarLabel}>Fleet Orchestration</div>
               <div className={styles.tableListScroll}>
                  {loading ? <div className={styles.sidebarLoading}>Syncing Schema...</div> : 
                   tables.map(table => (
                     <div key={table} className={styles.tableBtnWrapper}>
                       <button 
                         className={`${styles.tableBtn} ${selectedTable === table ? styles.tableActive : ""}`}
                         onClick={() => handleTableSelect(table)}
                       >
                          <span className={styles.tableIconMini}>#</span> {table}
                       </button>
                       <div className={styles.tableRowActions}>
                         <button className={styles.tableEditBtn} onClick={() => renameTable(table)}>✏️</button>
                         <button className={styles.tableDeleteBtn} onClick={() => dropTable(table)}>🗑️</button>
                       </div>
                     </div>
                   ))}
               </div>
            </div>
            {productSlug && (
               <div className={styles.sidebarBottom}>
                  <button className={styles.resetFleetBtn} onClick={resetFleet}>
                     <span className={styles.resetIcon}>⚡</span> Factory Reset Fleet
                  </button>
               </div>
            )}
         </aside>

         <main className={styles.studioMain}>
             <div className={styles.studioToolbox}>
                <div className={styles.toolboxLeft}>
                   <input 
                     type="text" 
                     placeholder="Search fleet data..." 
                     className={styles.studioSearch} 
                     value={searchTerm}
                     onChange={(e) => setSearchTerm(e.target.value)}
                   />
                   <button className={`${styles.toolBtn} ${isSelectMode ? styles.toolBtnActive : ""}`} onClick={() => setIsSelectMode(!isSelectMode)}>
                     <span>{isSelectMode ? "Cancel Select" : "Bulk Select"}</span>
                   </button>
                   {isSelectMode && selectedRowIds.size > 0 && (
                     <button className={`${styles.toolBtn} ${styles.btnDanger}`} onClick={deleteSelectedRows}>
                       Delete ({selectedRowIds.size})
                     </button>
                   )}
                </div>
             </div>

            <div className={styles.gridContainer}>
               {error ? <div className={styles.gridErrorState}><p>{error}</p></div> : 
                fetching ? <div className={styles.gridOverlay}><div className={styles.studioSpinner}></div></div> : 
                selectedTable ? (
                 <div className={styles.spreadsheet}>
                    <table className={styles.studioTable}>
                       <thead>
                          <tr>
                             <th className={styles.rowNumberCol}>{isSelectMode ? "✓" : "#"}</th>
                             {columns.map(col => (
                                <th key={col.name}>
                                   <div className={styles.thContent}>
                                      <div className={styles.thMain} onClick={() => handleSort(col.name)}>
                                        <span className={styles.thName}>{col.name}</span>
                                      </div>
                                      <div className={styles.colActions}>
                                        <button className={styles.miniColBtn} onClick={() => renameColumn(col.name)}>✏️</button>
                                        <button className={styles.miniColBtn} onClick={() => dropColumn(col.name)}>🗑️</button>
                                      </div>
                                      {col.pk === 1 && <span className={styles.pkBadge}>PK</span>}
                                   </div>
                                </th>
                             ))}
                          </tr>
                       </thead>
                       <tbody>
                          {filteredRows.length === 0 ? (
                            <tr><td colSpan={columns.length + 1} style={{textAlign:'center',padding:'100px',opacity:0.5,minWidth: columns.length * 180 + 'px'}}>No data in cluster</td></tr>
                          ) : (
                            filteredRows.map((row, idx) => (
                              <tr key={idx}>
                                <td className={styles.rowNumber}>
                                  {isSelectMode ? <input type="checkbox" checked={selectedRowIds.has(row[columns[0]?.name])} onChange={() => toggleRowSelection(row[columns[0]?.name])}/> : (idx + 1)}
                                </td>
                                {columns.map(col => (
                                  <td key={col.name} onDoubleClick={() => setEditingCell({ rowIdx: idx, colName: col.name, value: row[col.name] })}>
                                    <div className={styles.cellContent}>
                                      {editingCell?.rowIdx === idx && editingCell?.colName === col.name ? (
                                        <input autoFocus className={styles.cellInput} value={editingCell.value} onChange={(e) => setEditingCell({ ...editingCell, value: e.target.value })} onBlur={() => saveCellEdit(row, col.name, editingCell.value)}/>
                                      ) : (row[col.name] === null ? <span className={styles.nullCell}>NULL</span> : String(row[col.name]))}
                                    </div>
                                  </td>
                                ))}
                              </tr>
                            ))
                          )}
                       </tbody>
                    </table>
                 </div>
               ) : <div className={styles.noTableState}><h2>Select a table to begin fleet orchestration.</h2></div>}
            </div>

            <footer className={styles.studioFooter}>
               <div>Fleet Master Protocol v4.0.2</div>
               <div>Sync Status: Online</div>
            </footer>
         </main>
      </div>
    </div>
  );
}
