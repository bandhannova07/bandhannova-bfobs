"use client";
import React, { useState, useEffect, useMemo } from "react";
import styles from "../page.module.css";
import { fetchAPI } from "../../../../../lib/api";

interface ShardStudioProps {
  shard: {
    slug: string;
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
  const [error, setError] = useState<string | null>(null);

  // New UI States
  const [searchTerm, setSearchTerm] = useState("");
  const [sortConfig, setSortConfig] = useState<{ key: string; direction: "asc" | "desc" | null }>({ key: "", direction: null });
  const [isSelectMode, setIsSelectMode] = useState(false);
  const [selectedRowIds, setSelectedRowIds] = useState<Set<any>>(new Set());
  const [editingCell, setEditingCell] = useState<{ rowIdx: number; colName: string; value: any } | null>(null);

  useEffect(() => {
    loadTables();
  }, [shard.slug]);

  const execSQL = async (sql: string) => {
    return await fetchAPI(`/admin/db/execute`, {
      method: "POST",
      body: JSON.stringify({ shard: shard.slug, sql }),
    });
  };

  const loadTables = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await execSQL("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'");
      if (res.success && res.result) {
        const rows = res.result.rows || [];
        const tableNames = rows.map((t: any) => t.name || t.NAME || Object.values(t)[0]);
        setTables(tableNames as string[]);
        if (tableNames.length > 0 && !selectedTable) {
          handleTableSelect(tableNames[0] as string);
        }
      } else {
        setError(res.message || "No tables found");
      }
    } catch (err: any) {
      setError(err.message || "Connection failed");
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

      if (schemaRes.success && schemaRes.result) setColumns(schemaRes.result.rows || []);
      if (dataRes.success && dataRes.result) setRows(dataRes.result.rows || []);
    } catch (err: any) {
      setError(err.message || "Fetch failed");
    } finally {
      setFetching(false);
    }
  };

  // ─── Transformation Logic ───────────────────────
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

  // ─── Actions ────────────────────────────────────
  const toggleRowSelection = (id: any) => {
    const next = new Set(selectedRowIds);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setSelectedRowIds(next);
  };

  const deleteSelectedRows = async () => {
    if (selectedRowIds.size === 0 || !selectedTable) return;
    const primaryKey = columns.find(c => c.pk === 1)?.name || columns[0]?.name;
    if (!primaryKey) return alert("Cannot delete without a primary key identifier.");

    if (!confirm(`Are you sure you want to delete ${selectedRowIds.size} rows?`)) return;

    setFetching(true);
    try {
      const idsToDelete = Array.from(selectedRowIds).map(id => typeof id === 'string' ? `'${id}'` : id).join(",");
      const res = await execSQL(`DELETE FROM "${selectedTable}" WHERE "${primaryKey}" IN (${idsToDelete})`);
      if (res.success) {
        handleTableSelect(selectedTable);
      } else {
        alert(res.message);
      }
    } catch (err: any) {
      alert("Delete failed: " + err.message);
    } finally {
      setFetching(false);
    }
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
      } else {
        alert(res.message);
      }
    } catch (err: any) {
      alert("Update failed: " + err.message);
    } finally {
      setFetching(false);
    }
  };

  const handleSort = (key: string) => {
    setSortConfig(prev => ({
      key,
      direction: prev.key === key && prev.direction === "asc" ? "desc" : "asc"
    }));
  };

  const execBulkSQL = async (sql: string) => {
    const productSlug = shard.slug.split("-shard")[0];
    return await fetchAPI(`/admin/db/execute-bulk`, {
      method: "POST",
      body: JSON.stringify({ product_slug: productSlug, sql }),
    });
  };

  const dropTable = async (tableName: string) => {
    if (!confirm(`⚠️ FLEET ACTION: This will DROP table "${tableName}" across ALL shards. Proceed?`)) return;

    setFetching(true);
    try {
      const res = await execBulkSQL(`DROP TABLE "${tableName}"`);
      if (res.success) {
        setSelectedTable(null);
        loadTables();
      } else {
        alert("Fleet operation failed: " + res.message);
      }
    } catch (err: any) {
      alert("Drop failed: " + err.message);
    } finally {
      setFetching(false);
    }
  };

  const renameTable = async (oldName: string) => {
    const newName = prompt("Enter new name for table:", oldName);
    if (!newName || newName === oldName) return;

    if (!confirm(`⚠️ FLEET ACTION: Rename "${oldName}" to "${newName}" across ALL shards?`)) return;

    setFetching(true);
    try {
      const res = await execBulkSQL(`ALTER TABLE "${oldName}" RENAME TO "${newName}"`);
      if (res.success) {
        setSelectedTable(newName);
        loadTables();
      } else {
        alert(res.message);
      }
    } catch (err: any) {
      alert("Rename failed: " + err.message);
    } finally {
      setFetching(false);
    }
  };

  const dropColumn = async (colName: string) => {
    if (!selectedTable) return;
    if (!confirm(`⚠️ FLEET ACTION: Drop column "${colName}" from "${selectedTable}" across ALL shards?`)) return;

    setFetching(true);
    try {
      const res = await execBulkSQL(`ALTER TABLE "${selectedTable}" DROP COLUMN "${colName}"`);
      if (res.success) {
        handleTableSelect(selectedTable);
      } else {
        alert(res.message);
      }
    } catch (err: any) {
      alert("Drop column failed: " + err.message);
    } finally {
      setFetching(false);
    }
  };

  const renameColumn = async (oldCol: string) => {
    if (!selectedTable) return;
    const newCol = prompt("Enter new name for column:", oldCol);
    if (!newCol || newCol === oldCol) return;

    if (!confirm(`⚠️ FLEET ACTION: Rename column "${oldCol}" to "${newCol}" across ALL shards?`)) return;

    setFetching(true);
    try {
      const res = await execBulkSQL(`ALTER TABLE "${selectedTable}" RENAME COLUMN "${oldCol}" TO "${newCol}"`);
      if (res.success) {
        handleTableSelect(selectedTable);
      } else {
        alert(res.message);
      }
    } catch (err: any) {
      alert("Rename column failed: " + err.message);
    } finally {
      setFetching(false);
    }
  };

  const resetFleet = async () => {
    const infraId = prompt("⚠️ CRITICAL ACTION: Enter Infrastructure ID to factory reset ALL shards for this product:");
    if (!infraId) return;

    if (!confirm("ARE YOU ABSOLUTELY SURE? This will DELETE ALL DATA across the entire fleet and cannot be undone!")) return;

    setFetching(true);
    try {
      // We send the infraId for verification on backend
      const res = await fetchAPI(`/admin/db/reset-fleet`, {
        method: "POST",
        body: JSON.stringify({
          product_slug: shard.slug.split("-shard")[0],
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
          <img src="/favicon.ico" alt="BN" className={styles.studioFavicon} />
          <div className={styles.studioBreadcrumbs}>
            <span className={styles.crumbProject}>Fleet Mode</span>
            <span className={styles.crumbDivider}>/</span>
            <span className={styles.crumbShard}>{shard.name}</span>
            {selectedTable && (
              <><span className={styles.crumbDivider}>/</span><span className={styles.crumbTable}>{selectedTable}</span></>
            )}
          </div>
        </div>
        <div className={styles.studioTopActions}>
          <div className={styles.connectionStatus}>
            <div className={error ? styles.deadPulse : styles.livePulse}></div>
            <span>{error ? "FLEET OFFLINE" : "FLEET SYNC ACTIVE"}</span>
          </div>
          <button className={styles.exitStudioSimple} onClick={onClose}>
            Exit Studio
          </button>
        </div>
      </header>

      <div className={styles.studioLayout}>
        <aside className={styles.studioSidebar}>
          <div className={styles.sidebarTop}>
            <div className={styles.sidebarSection}>
              <div className={styles.sidebarLabel}>ORCHESTRATION</div>
              <div className={styles.sidebarNav}>
                <button className={`${styles.navItem} ${styles.navActive}`}>
                  <span className={styles.navIcon}>🌐</span> Fleet Tables
                </button>
              </div>
            </div>

            <div className={styles.sidebarSection}>
              <div className={styles.sidebarLabel}>ALL TABLES</div>
              <div className={styles.tableListScroll}>
                {loading ? (
                  <div className={styles.sidebarLoading}>Loading fleet schema...</div>
                ) : tables.length === 0 ? (
                  <div className={styles.sidebarLoading}>No tables found</div>
                ) : tables.map(table => (
                  <div key={table} className={styles.tableBtnWrapper}>
                    <button
                      className={`${styles.tableBtn} ${selectedTable === table ? styles.tableActive : ""}`}
                      onClick={() => handleTableSelect(table)}
                    >
                      <span className={styles.tableIconMini}>#</span> {table}
                    </button>
                    <div className={styles.tableRowActions}>
                      <button className={styles.tableEditBtn} onClick={() => renameTable(table)} title="Rename Table">✏️</button>
                      <button className={styles.tableDeleteBtn} onClick={() => dropTable(table)} title="Drop Table">🗑️</button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div className={styles.sidebarBottom}>
            <button className={styles.resetFleetBtn} onClick={resetFleet}>
              <span className={styles.resetIcon}>⚡</span> Reset All Shards
            </button>
          </div>
        </aside>

        <main className={styles.studioMain}>
          <div className={styles.studioToolbox}>
            <div className={styles.toolboxLeft}>
              <div className={styles.searchWrapper}>
                <span className={styles.searchIcon}>🔍</span>
                <input
                  type="text"
                  placeholder="Search fleet data..."
                  className={styles.studioSearch}
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
              <button
                className={`${styles.toolBtn} ${isSelectMode ? styles.toolBtnActive : ""}`}
                onClick={() => setIsSelectMode(!isSelectMode)}
              >
                <span>{isSelectMode ? "Cancel Select" : "Bulk Select Rows"}</span>
              </button>
              {isSelectMode && selectedRowIds.size > 0 && (
                <button className={`${styles.toolBtn} ${styles.btnDanger}`} onClick={deleteSelectedRows}>
                  <span>Delete Rows ({selectedRowIds.size})</span>
                </button>
              )}
            </div>
            <div className={styles.toolboxRight}>
              <span className={styles.shardIdLabel}>Shard: {shard.slug}</span>
            </div>
          </div>

          <div className={styles.gridContainer}>
            {error ? (
              <div className={styles.gridErrorState}>
                <div className={styles.bigStudioIcon}>⚠️</div>
                <p>{error}</p>
                <button className="btn btn-glass" style={{ marginTop: '12px' }} onClick={loadTables}>Retry Sync</button>
              </div>
            ) : fetching ? (
              <div className={styles.gridOverlay}>
                <div className={styles.studioSpinner}></div>
                <p>Orchestrating fleet-wide changes...</p>
              </div>
            ) : selectedTable ? (
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
                              {sortConfig.key === col.name && (
                                <span className={styles.sortArrow}>{sortConfig.direction === "asc" ? "↑" : "↓"}</span>
                              )}
                            </div>
                            <div className={styles.colActions}>
                              <button className={styles.miniColBtn} onClick={() => renameColumn(col.name)} title="Rename Column">✏️</button>
                              <button className={styles.miniColBtn} onClick={() => dropColumn(col.name)} title="Drop Column">🗑️</button>
                            </div>
                            {col.pk === 1 && <span className={styles.pkBadge}>PK</span>}
                          </div>
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {filteredRows.length === 0 ? (
                      <tr>
                        <td
                          colSpan={columns.length + 1}
                          style={{
                            textAlign: 'center',
                            padding: '100px',
                            opacity: 0.5,
                            fontSize: '18px',
                            minWidth: columns.length * 180 + 'px'
                          }}
                        >
                          No matching fleet data in this shard
                        </td>
                      </tr>
                    ) : (
                      filteredRows.map((row, idx) => {
                        const pkName = columns.find(c => c.pk === 1)?.name || columns[0]?.name;
                        const rowId = row[pkName];
                        return (
                          <tr key={idx} className={selectedRowIds.has(rowId) ? styles.rowSelected : ""}>
                            <td className={styles.rowNumber}>
                              {isSelectMode ? (
                                <input
                                  type="checkbox"
                                  style={{ height: '16px', width: '16px' }}
                                  checked={selectedRowIds.has(rowId)}
                                  onChange={() => toggleRowSelection(rowId)}
                                />
                              ) : (idx + 1)}
                            </td>
                            {columns.map(col => (
                              <td
                                key={col.name}
                                onDoubleClick={() => setEditingCell({ rowIdx: idx, colName: col.name, value: row[col.name] })}
                              >
                                <div className={styles.cellContent}>
                                  {editingCell?.rowIdx === idx && editingCell?.colName === col.name ? (
                                    <input
                                      autoFocus
                                      className={styles.cellInput}
                                      value={editingCell.value}
                                      onChange={(e) => setEditingCell({ ...editingCell, value: e.target.value })}
                                      onBlur={() => saveCellEdit(row, col.name, editingCell.value)}
                                      onKeyDown={(e) => {
                                        if (e.key === "Enter") saveCellEdit(row, col.name, editingCell.value);
                                        if (e.key === "Escape") setEditingCell(null);
                                      }}
                                    />
                                  ) : (
                                    row[col.name] === null ? <span className={styles.nullCell}>NULL</span> : String(row[col.name])
                                  )}
                                </div>
                              </td>
                            ))}
                          </tr>
                        );
                      })
                    )}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className={styles.noTableState}>
                <div className={styles.bigStudioIcon}>🌐</div>
                <h2>Fleet Orchestrator Ready</h2>
                <p>Select a table to begin multi-shard synchronization.</p>
              </div>
            )}
          </div>

          <footer className={styles.studioFooter}>
            <div className={styles.footerInfo}>
              {selectedTable && !error && (
                <>
                  <span>Shard Rows: {filteredRows.length}</span>
                  <span className={styles.footerSep}>|</span>
                  <span>Sync Nodes: Active</span>
                </>
              )}
            </div>
            <div className={styles.footerBrand}>BandhanNova Fleet Studio v3.1</div>
          </footer>
        </main>
      </div>
    </div>
  );
}
