"use client";

import React, { useState, useEffect } from "react";
import styles from "../page.module.css";
import { fetchAPI } from "../../../../../lib/api";
import { API_URL } from "../../../../../lib/constants";

interface Product {
  id: string;
  name: string;
  slug: string;
}

interface Bucket {
  id: string;
  name: string;
  slug: string;
  description: string;
  is_public: boolean;
  created_at: number;
}

interface FileInfo {
  name: string;
  path: string;
  size: number;
  url: string;
  last_modified?: string;
}

interface StorageViewProps {
  product: Product;
}

export default function StorageView({ product }: StorageViewProps) {
  const [buckets, setBuckets] = useState<Bucket[]>([]);
  const [loading, setLoading] = useState(true);
  const [hasDatabase, setHasDatabase] = useState(false);
  const [activeBucket, setActiveBucket] = useState<Bucket | null>(null);
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [loadingFiles, setLoadingFiles] = useState(false);

  // Modals
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState<Bucket | null>(null);
  const [confirmText, setConfirmText] = useState("");
  const [masterKey, setMasterKey] = useState("");

  // New Bucket Form
  const [newBucket, setNewBucket] = useState({ name: "", slug: "", description: "", is_public: true });

  // Upload
  const [uploading, setUploading] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploadStatus, setUploadStatus] = useState<string | null>(null);

  useEffect(() => {
    checkDatabaseAndFetchBuckets();
  }, [product.slug]);

  const checkDatabaseAndFetchBuckets = async () => {
    setLoading(true);
    try {
      // 1. Check if database shards exist (minimal check)
      const dbData = await fetchAPI(`/admin/databases?product_id=${product.id}`);
      setHasDatabase(dbData.success && dbData.databases && dbData.databases.length > 0);

      // 2. Fetch buckets
      await fetchBuckets();
    } catch (err) {
      console.error("Failed to sync storage infrastructure", err);
    } finally {
      setLoading(false);
    }
  };

  const fetchBuckets = async () => {
    try {
      const data = await fetchAPI(`/storage/p/${product.slug}/buckets`);
      if (data.success) setBuckets(data.buckets || []);
    } catch (err) {
      console.error("Failed to fetch buckets", err);
    }
  };

  const handleCreateBucket = async () => {
    try {
      const data = await fetchAPI(`/storage/p/${product.slug}/buckets`, {
        method: "POST",
        body: JSON.stringify({
          name: newBucket.name,
          slug: newBucket.slug || newBucket.name.toLowerCase().replace(/\s+/g, '-')
        })
      });
      if (data.success) {
        setShowCreateModal(false);
        setNewBucket({ name: "", slug: "", description: "", is_public: true });
        fetchBuckets();
      } else {
        alert(data.message);
      }
    } catch (err: any) {
      alert("Failed to create bucket: " + err.message);
    }
  };

  const handleDeleteBucket = async () => {
    if (!showDeleteModal) return;
    const expected = `I am Bandhan, I want to delete this bucket named ${showDeleteModal.name}.`;
    if (confirmText !== expected) {
      alert("Confirmation text mismatch!");
      return;
    }

    try {
      const data = await fetchAPI(`/storage/buckets/${showDeleteModal.id}?confirm=${encodeURIComponent(confirmText)}`, {
        method: "DELETE",
        headers: {
          "X-BandhanNova-Master-Key": masterKey
        }
      });
      if (data.success) {
        setShowDeleteModal(null);
        setConfirmText("");
        setMasterKey("");
        fetchBuckets();
      } else {
        alert(data.message);
      }
    } catch (err: any) {
      alert("Delete failed: " + err.message);
    }
  };

  const openBucket = async (bucket: Bucket) => {
    setActiveBucket(bucket);
    setLoadingFiles(true);
    setFiles([]);
    try {
      const data = await fetchAPI(`/storage/p/${product.slug}/b/${bucket.slug}/files`);
      if (data.success) setFiles(data.files || []);
    } catch (err) {
      console.error("Failed to fetch files", err);
    } finally {
      setLoadingFiles(false);
    }
  };

  const handleUpload = async () => {
    if (!selectedFile || !activeBucket) return;

    setUploading(true);
    setUploadStatus("Uploading...");

    const formData = new FormData();
    formData.append("file", selectedFile);
    formData.append("product_slug", product.slug);
    formData.append("bucket", activeBucket.slug);

    try {
      const token = sessionStorage.getItem("admin_token");
      const res = await fetch(`${API_URL}/storage/upload`, {
        method: "POST",
        headers: { "X-Admin-Token": token || "" },
        body: formData
      });

      const data = await res.json();
      if (data.success) {
        setUploadStatus("✅ Done");
        setSelectedFile(null);
        openBucket(activeBucket);
        setTimeout(() => setUploadStatus(null), 2000);
      } else {
        setUploadStatus("❌ Failed");
      }
    } catch (err) {
      setUploadStatus("❌ Error");
    } finally {
      setUploading(false);
    }
  };

  const handleDeleteFile = async (fileName: string) => {
    if (!activeBucket || !confirm(`Delete ${fileName}?`)) return;

    try {
      const data = await fetchAPI(`/storage/p/${product.slug}/b/${activeBucket.slug}/f/${fileName}`, {
        method: "DELETE"
      });
      if (data.success) {
        openBucket(activeBucket);
      } else {
        alert("Delete failed");
      }
    } catch (err: any) {
      alert("Error deleting file: " + err.message);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    // Minimal feedback
  };

  if (loading) return <div className={styles.loading}>Syncing Storage Fleet...</div>;

  return (
    <div className={styles.tabContent}>
      <div className={styles.sectionHeader}>
        <div>
          <h2 className={styles.sectionTitle}>Storage Explorer</h2>
          <p className={styles.sectionSubtitle}>Manage assets for {product.name}</p>
        </div>
        <button
          className="btn btn-primary"
          onClick={() => setShowCreateModal(true)}
          disabled={!hasDatabase}
        >
          + New Bucket
        </button>
      </div>

      <div className={styles.bucketGrid}>
        {buckets.map(b => (
          <div key={b.id} className={styles.minimalCard} onClick={() => openBucket(b)}>
            <div className={styles.cardIcon}>📦</div>
            <div className={styles.cardInfo}>
              <h3>{b.name}</h3>
              <p>/{b.slug}</p>
            </div>
            <button
              className={styles.cardActionBtn}
              onClick={(e) => { e.stopPropagation(); setShowDeleteModal(b); }}
            >
              🗑️
            </button>
            <button
              className="btn btn-glass"
              style={{position: 'absolute', bottom: '12px', left: '12px', right: '12px', fontSize: '11px'}}
              onClick={() => openBucket(b)}
            >
              Inspect Bucket</button>
          </div>
        ))}
      </div>

      {/* Centered File Explorer Overlay */}
      {activeBucket && (
        <div className={styles.overlay}>
          <div className={styles.sidePanel}>
            <div className={styles.panelHeader}>
              <h3>{activeBucket.name}</h3>
              <button className={styles.closeBtn} onClick={() => setActiveBucket(null)}>✕</button>
            </div>

            <div className={styles.panelBody}>
              {/* Simple Upload */}
              <div className={styles.uploadArea}>
                <input
                  type="file"
                  id="file-up"
                  className={styles.hiddenInput}
                  onChange={(e) => setSelectedFile(e.target.files ? e.target.files[0] : null)}
                />
                <label htmlFor="file-up" className={styles.minimalDropzone}>
                  {selectedFile ? selectedFile.name : "Choose File"}
                </label>
                {selectedFile && (
                  <button className="btn btn-primary" onClick={handleUpload} disabled={uploading}>
                    {uploading ? "..." : "Upload Asset"}
                  </button>
                )}
                {uploadStatus && <p className={styles.statusText}>{uploadStatus}</p>}
              </div>

              <div className={styles.fileListMinimal}>
                {loadingFiles ? (
                  <p>Syncing bucket data...</p>
                ) : files.length === 0 ? (
                  <p style={{textAlign: 'center', padding: '40px', opacity: 0.5}}>No assets found in this bucket.</p>
                ) : (
                  files.map(f => (
                    <div key={f.path} className={styles.fileRowMinimal}>
                      <div className={styles.fileMain}>
                        <span className={styles.fileIcon}>📄</span>
                        <div className={styles.fileDetails}>
                          <span className={styles.fileName}>{f.name}</span>
                          <span className={styles.fileSize}>{(f.size / 1024).toFixed(1)} KB</span>
                        </div>
                      </div>
                      <div className={styles.fileActionsMinimal}>
                        <button onClick={() => copyToClipboard(f.url)} className={styles.fileActionsMinimalBtn} title="Copy URL">Copy URL</button>
                        <button onClick={() => window.open(f.url, '_blank')} className={styles.fileActionsMinimalBtn} title="View">View</button>
                        <button onClick={() => handleDeleteFile(f.name)} className={styles.deleteText} title="Delete Asset">🗑️</button>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Create Bucket Modal */}
      {showCreateModal && (
        <div className={styles.modalOverlay}>
          <div className={styles.minimalModal}>
            <h3>Provision New Bucket</h3>
            <div className={styles.minimalField}>
              <label>Bucket Display Name</label>
              <input
                className={styles.confirmInput}
                value={newBucket.name}
                onChange={e => setNewBucket({ ...newBucket, name: e.target.value })}
                placeholder="e.g. Media Assets"
              />
            </div>
            <div className={styles.minimalField}>
              <label>Bucket Slug</label>
              <input
                className={styles.confirmInput}
                value={newBucket.slug}
                onChange={e => setNewBucket({ ...newBucket, slug: e.target.value })}
                placeholder="e.g. media-assets"
              />
            </div>
            <div className={styles.modalFooter}>
              <button onClick={() => setShowCreateModal(false)}>Discard</button>
              <button className="btn btn-primary" onClick={handleCreateBucket}>Create Bucket</button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Bucket Modal */}
      {showDeleteModal && (
        <div className={styles.modalOverlay}>
          <div className={styles.minimalModal}>
            <h3 style={{ color: "var(--neon-red)" }}>Decommission Bucket</h3>
            <p>This will remove <strong>{showDeleteModal.name}</strong> and all its assets permanently.</p>

            <div className={styles.minimalField}>
              <label>Infrastructure Master Key</label>
              <input
                className={styles.confirmInput}
                type="password"
                value={masterKey}
                onChange={e => setMasterKey(e.target.value)}
              />
            </div>

            <div className={styles.minimalField}>
              <label>Type phrase to confirm</label>
              <p className={styles.hint}>I am Bandhan, I want to delete this bucket named {showDeleteModal.name}.</p>
              <input
                className={styles.confirmInput}
                value={confirmText}
                onChange={e => setConfirmText(e.target.value)}
              />
            </div>

            <div className={styles.modalFooter}>
              <button onClick={() => setShowDeleteModal(null)}>Cancel</button>
              <button className="btn btn-primary" style={{ background: "var(--neon-red)" }} onClick={handleDeleteBucket}>Confirm Destruction</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
