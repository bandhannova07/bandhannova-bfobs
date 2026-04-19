"use client";

import { useState, useEffect } from "react";
import { fetchAPI } from "../lib/api";
import styles from "./PulseHealth.module.css";

export default function PulseHealth() {
  const [pulse, setPulse] = useState(null);
  const [loading, setLoading] = useState(true);

  const loadPulse = async () => {
    try {
      const res = await fetchAPI("/admin/health/pulse");
      if (res.success) setPulse(res.pulse);
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

  if (loading) return <div className={styles.pulseContainer}>Scanning Ecosystem Heartbeat...</div>;
  if (!pulse) return null;

  const shards = Object.values(pulse);

  return (
    <div className={styles.pulseContainer}>
      <div className={styles.pulseHeader}>
        <div className={styles.pulseDot}></div>
        <span className={styles.pulseTitle}>Ecosystem Heartbeat (Live Pulse)</span>
        <span className={styles.pulseTime}>Interval: 3m</span>
      </div>
      <div className={styles.pulseGrid}>
        {shards.map((shard) => (
          <div key={shard.name} className={styles.pulseItem}>
            <div className={styles.shardInfo}>
              <span className={styles.shardName}>{shard.name}</span>
              <span className={styles.shardType}>{shard.type}</span>
            </div>
            <div className={styles.shardStatus}>
              <span className={`${styles.statusDot} ${styles[shard.status]}`}></span>
              <span className={styles.latency}>{(shard.latency / 1000000).toFixed(1)}ms</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
