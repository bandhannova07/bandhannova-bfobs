"use client";

import React, { useState, useEffect } from "react";
import { fetchAPI } from "../lib/api";
import styles from "./PulseHealth.module.css";

interface Shard {
  name: string;
  type: string;
  status: "healthy" | "degraded" | "offline";
  latency: number;
}

export default function PulseHealth() {
  const [shards, setShards] = useState<Shard[]>([]);
  const [loading, setLoading] = useState(true);

  const loadPulse = async () => {
    try {
      const res = await fetchAPI("/admin/health/pulse");
      if (res.success && res.pulse) {
        setShards(Object.values(res.pulse) as Shard[]);
      }
    } catch (err) {
      console.error("Pulse error", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadPulse();
    const interval = setInterval(loadPulse, 180000); // 3 minutes
    return () => clearInterval(interval);
  }, []);

  if (loading) return <div className={styles.pulseContainer}>SYNCING ECOSYSTEM HEARTBEAT...</div>;
  if (shards.length === 0) return null;

  return (
    <div className={`glass-panel ${styles.pulseContainer}`}>
      <div className={styles.pulseHeader}>
        <div className={styles.pulseDot}></div>
        <span className={styles.pulseTitle}>Fleet Heartbeat (L-Pulse)</span>
        <span className={styles.pulseTime}>Interval: 180s</span>
      </div>
      <div className={styles.pulseGrid}>
        {shards.map((shard) => (
          <div key={shard.name} className={styles.pulseItem}>
            <div className={styles.shardInfo}>
              <span className={styles.shardName}>{shard.name}</span>
              <span className={styles.shardType}>{shard.type.toUpperCase()}</span>
            </div>
            <div className={styles.shardStatus}>
              <div className={`${styles.statusPill} ${styles[shard.status]}`}>
                <span className={styles.statusDot}></span>
                <span className={styles.latency}>{(shard.latency / 1000000).toFixed(1)}ms</span>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

