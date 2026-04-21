"use client";

import React, { useState, useEffect } from "react";
import styles from "../page.module.css";
import { API_URL } from "@/lib/constants";

interface Product {
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
  path: string;
  type: string;
  size?: number;
  lastCommit?: {
    date: string;
  };
}

interface StorageViewProps {
  product: Product;
}

export default function StorageView({ product }: StorageViewProps) {
  const [buckets, setBuckets] = useState<Bucket[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeBucket, setActiveBucket] = useState<Bucket | null>(null);
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [loadingFiles, setLoadingFiles] = useState(false);

  // Modals
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState<Bucket | null>(null);
  const [confirmText, setConfirmText] = useState("");
  const [masterKey, setMasterKey] = useState("");

  // New Bucket Form
  const [newBucket, setNewBucket] = useState({ name: "", description: "", is_public: false });

  // Upload
  const [uploading, setUploading] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploadStatus, setUploadStatus] = useState<string | null>(null);

  useEffect(() => {
    fetchBuckets();
  }, [product.slug]);

  const fetchBuckets = async () => {
    try {
      const token = sessionStorage.getItem("admin_token");
      const res = await fetch(`${API_URL}/storage/p/${product.slug}/buckets`, {
        headers: { "X-Admin-Token": token || "" }
      });
      const data = await res.json();
      if (data.success) setBuckets(data.buckets || []);
    } catch (err) {
      console.error("Failed to fetch buckets", err);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateBucket = async () => {
    try {
      const token = sessionStorage.getItem("admin_token");
      const res = await fetch(`${API_URL}/storage/p/${product.slug}/buckets`, {
        method: "POST",
        headers: {
          "X-Admin-Token": token || "",
          "Content-Type": "application/json"
        },
        body: JSON.stringify(newBucket)
      });
      const data = await res.json();
      if (data.success) {
        setShowCreateModal(false);
        setNewBucket({ name: "", description: "", is_public: false });
        fetchBuckets();
      } else {
        alert(data.message + (data.details ? ": " + data.details : ""));
      }
    } catch (err) {
      alert("Failed to create bucket");
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
      const token = sessionStorage.getItem("admin_token");
      const res = await fetch(`${API_URL}/storage/buckets/${showDeleteModal.id}?confirm=${encodeURIComponent(confirmText)}`, {
        method: "DELETE",
        headers: {
          "X-Admin-Token": token || "",
          "X-BandhanNova-Master-Key": masterKey
        }
      });
      const data = await res.json();
      if (data.success) {
        setShowDeleteModal(null);
        setConfirmText("");
        setMasterKey("");
        fetchBuckets();
      } else {
        alert(data.message);
      }
    } catch (err) {
      alert("Delete failed");
    }
  };

  const openBucket = async (bucket: Bucket) => {
    setActiveBucket(bucket);
    setLoadingFiles(true);
    setFiles([]);
    try {
      const token = sessionStorage.getItem("admin_token");
      const res = await fetch(`${API_URL}/storage/p/${product.slug}/b/${bucket.slug}/files`, {
        headers: { "X-Admin-Token": token || "" }
      });
      const data = await res.json();
      if (data.success) setFiles(data.files || []);
    } catch (err) {
      console.error("Failed to fetch files", err);
    } finally {
      setLoadingFiles(false);
    }
  };

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedFile || !activeBucket) return;

    setUploading(true);
    setUploadStatus("Pushing to HF " + activeBucket.slug + "...");

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
        setUploadStatus("✅ Upload Successful!");
        setSelectedFile(null);
        openBucket(activeBucket); // Refresh file list
      } else {
        setUploadStatus("❌ Upload Failed: " + (data.message || "Unknown error"));
      }
    } catch (err) {
      setUploadStatus("❌ Network Error connecting to gateway");
    } finally {
      setUploading(false);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    alert("Copied: " + text);
  };

  if (loading) return <div className={styles.loading}>Initializing Fleet Storage...</div>;

  return (
    <div className={styles.tabContent}>
      <div className={styles.sectionHeader}>
        <h2 className={styles.sectionTitle}>Managed Storage Buckets</h2>
        <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
          + New Bucket
        </button>
      </div>

      <div className={styles.bucketGrid}>
        {buckets.map(b => (
          <div key={b.id} className={styles.bucketCard}>
            <button className={styles.deleteBucketBtn} onClick={() => setShowDeleteModal(b)}>🗑️</button>
            <div className={styles.bucketIcon}>{b.is_public ? "🌍" : "🔒"}</div>
            <div className={styles.bucketInfo}>
              <h4>{b.name}</h4>
              <p>{b.description || "No description provided."}</p>
            </div>
            <div className={styles.bucketMeta}>
              <span className={`${styles.badge} ${b.is_public ? styles.publicBadge : styles.privateBadge}`}>
                {b.is_public ? "Public" : "Private"}
              </span>
              <code>/{b.slug}</code>
            </div>
            <div className={styles.bucketActions}>
              <button className={styles.actionBtn} onClick={() => copyToClipboard(`${API_URL}/storage/upload?bucket=${b.slug}`)}>
                🔗 Upload URL
              </button>
              <button className={styles.actionBtn} onClick={() => copyToClipboard(`${window.location.origin.replace('3000', '8080')}/storage/view/${product.slug}/${b.slug}/{file}`)}>
                🖼️ View URL
              </button>
              <button className={`${styles.actionBtn} btn-primary`} style={{ gridColumn: "span 2", marginTop: "5px" }} onClick={() => openBucket(b)}>
                📂 Open Bucket
              </button>
            </div>
          </div>
        ))}
      </div>

      {/* File Explorer Overlay */}
      {activeBucket && (
        <div className={styles.explorerOverlay}>
          <div className={styles.explorerContent}>
            <div className={styles.explorerHeader}>
              <h3>📂 {activeBucket.name} <small style={{ opacity: 0.5, fontSize: "12px" }}>/{activeBucket.slug}</small></h3>
              <button className="btn btn-secondary" onClick={() => setActiveBucket(null)}>Close</button>
            </div>

            <div className={styles.explorerContent} style={{ padding: "32px", background: "rgba(255,255,255,0.01)" }}>
              <form onSubmit={handleUpload} className={styles.uploadForm} style={{ flexDirection: "row", alignItems: "center" }}>
                <input
                  type="file"
                  id="file-explorer-upload"
                  className={styles.hiddenInput}
                  onChange={(e) => setSelectedFile(e.target.files ? e.target.files[0] : null)}
                />
                <label htmlFor="file-explorer-upload" className={styles.dropZone} style={{ height: "60px", flex: 1, marginBottom: 0 }}>
                  {selectedFile ? selectedFile.name : "Select File to Push"}
                </label>
                <button type="submit" className="btn btn-primary" disabled={uploading || !selectedFile} style={{ height: "60px", padding: "0 40px" }}>
                  {uploading ? "..." : "Push File"}
                </button>
              </form>
              {uploadStatus && <div className={styles.statusMsg}>{uploadStatus}</div>}

              <div className={styles.fileList}>
                {loadingFiles ? (
                  <div className={styles.loading}>Scanning HF Repository...</div>
                ) : files.length === 0 ? (
                  <div className={styles.emptyExplorer}>This bucket is empty.</div>
                ) : (
                  files.filter(f => !f.path.endsWith('.keep')).map(f => (
                    <div key={f.path} className={styles.fileRow}>
                      <div className={styles.fileIcon}>{f.type === 'file' ? "📄" : "📁"}</div>
                      <div className={styles.fileName}>{f.path.split('/').pop()}</div>
                      <div className={styles.fileSize}>{f.size ? (f.size / 1024).toFixed(1) + " KB" : "-"}</div>
                      <div className={styles.fileDate}>{f.lastCommit?.date ? new Date(f.lastCommit.date).toLocaleDateString() : "-"}</div>
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
        <div className={styles.modal}>
          <div className={styles.modalContent}>
            <h3>Create New Bucket</h3>
            <p>Define a new logical storage container.</p>
            <div className={styles.field}>
              <label>Bucket Name</label>
              <input
                className={styles.confirmInput}
                placeholder="e.g. User Avatars"
                value={newBucket.name}
                onChange={e => setNewBucket({ ...newBucket, name: e.target.value })}
              />
            </div>
            <div className={styles.field}>
              <label>Description</label>
              <input
                className={styles.confirmInput}
                placeholder="What is this bucket for?"
                value={newBucket.description}
                onChange={e => setNewBucket({ ...newBucket, description: e.target.value })}
              />
            </div>
            <div className={styles.modalActions}>
              <button className="btn btn-secondary" onClick={() => setShowCreateModal(false)}>Cancel</button>
              <button className="btn btn-primary" onClick={handleCreateBucket}>Create Bucket</button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Bucket Modal */}
      {showDeleteModal && (
        <div className={styles.modal}>
          <div className={styles.modalContent}>
            <h3 style={{ color: "var(--neon-red)" }}>Decommission Bucket</h3>
            <p>This action is permanent. All files in <strong>{showDeleteModal.name}</strong> will remain on HF but the bucket record will be removed.</p>

            <div className={styles.field}>
              <label>Master Key</label>
              <input
                type="password"
                className={styles.confirmInput}
                placeholder="Enter BandhanNova Master Key"
                value={masterKey}
                onChange={e => setMasterKey(e.target.value)}
              />
            </div>

            <div className={styles.field}>
              <label>Confirmation Phrase</label>
              <p style={{ fontSize: "11px", marginBottom: "8px" }}>Type: <code>I am Bandhan, I want to delete this bucket named {showDeleteModal.name}.</code></p>
              <input
                className={styles.confirmInput}
                placeholder="Type the phrase exactly"
                value={confirmText}
                onChange={e => setConfirmText(e.target.value)}
              />
            </div>

            <div className={styles.modalActions}>
              <button className="btn btn-secondary" onClick={() => setShowDeleteModal(null)}>Cancel</button>
              <button className="btn btn-primary" style={{ background: "var(--neon-red)" }} onClick={handleDeleteBucket}>
                Confirm Deletion
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
