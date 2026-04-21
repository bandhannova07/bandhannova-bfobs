"use client";

import React, { useState } from "react";
import styles from "../page.module.css";

import { API_URL } from "../../../../lib/constants";

interface Product {
  name: string;
  slug: string;
}

interface StorageViewProps {
  product: Product;
}

export default function StorageView({ product }: StorageViewProps) {
  const [uploading, setUploading] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploadStatus, setUploadStatus] = useState<string | null>(null);

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedFile) return;

    setUploading(true);
    setUploadStatus("Pushing to Hugging Face LFS...");

    const formData = new FormData();
    formData.append("file", selectedFile);
    formData.append("product_slug", product.slug);

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
      } else {
        setUploadStatus("❌ Upload Failed: " + (data.message || "Unknown error"));
      }
    } catch (err) {
      setUploadStatus("❌ Network Error connecting to gateway");
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className={styles.tabContent}>
      <div className={styles.storageLayout}>
         <div className={`glass-panel ${styles.uploadCard}`}>
            <h3>Cloud Asset Upload</h3>
            <p>Upload images, videos, and binaries for <strong>{product.name}</strong>.</p>
            <form onSubmit={handleUpload} className={styles.uploadForm}>
              <input 
                type="file" 
                id="file-upload" 
                className={styles.hiddenInput} 
                onChange={(e) => setSelectedFile(e.target.files ? e.target.files[0] : null)}
              />
              <label htmlFor="file-upload" className={styles.dropZone}>
                {selectedFile ? selectedFile.name : "Click or Drag to Upload"}
              </label>
              <button type="submit" className="btn btn-primary" disabled={uploading || !selectedFile}>
                 {uploading ? "Uploading..." : "Start Upload"}
              </button>
            </form>
            {uploadStatus && <div className={styles.statusMsg}>{uploadStatus}</div>}
         </div>

         <div className={`glass-panel ${styles.infoCard}`}>
            <h4>Dataset Details</h4>
            <div className={styles.detailsList}>
               <div className={styles.detail}>
                  <span>Repository</span>
                  <code>lordbandhan07/api-hunter-storage</code>
               </div>
               <div className={styles.detail}>
                  <span>Product Root</span>
                  <code>/{product.slug}/uploads/</code>
               </div>
            </div>
         </div>
      </div>
    </div>
  );
}

