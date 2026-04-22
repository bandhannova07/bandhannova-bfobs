import React from "react";

export default function ShowcasePage() {
  return (
    <main>
      <div className="glow-bg"></div>
      
      <div className="main-container">
        {/* Navigation */}
        <header className="nav">
          <div className="logo">BANDHAN<span>NOVA</span> BFOBS</div>
          <div style={{display:'flex', gap:'30px', fontSize:'13px', fontWeight:600, color:'var(--text-dim)'}}>
            <span style={{cursor:'pointer'}}>ARCHITECTURE</span>
            <span style={{cursor:'pointer'}}>ECOSYSTEM</span>
            <span style={{cursor:'pointer'}}>SECURITY</span>
          </div>
        </header>

        {/* Hero Section */}
        <section className="hero">
          <span className="hero-tag">Independent Software</span>
          <h1 className="hero-title">Infrastructure that <br/>Scales with Vision.</h1>
          <p className="hero-desc">
            The core engine powering the entire BandhanNova ecosystem. 
            An independent, high-performance sharded backend designed for absolute data isolation and infinite scale.
          </p>
        </section>

        {/* Feature Grid */}
        <section className="feature-grid">
          <div className="feature-card">
            <div className="card-icon">⚡</div>
            <h3 className="card-title">Sharded Fleet</h3>
            <p className="card-text">
              Every platform in our ecosystem gets its own dedicated Turso shards, orchestrated by a global master layer.
            </p>
          </div>

          <div className="feature-card">
            <div className="card-icon">☁️</div>
            <h3 className="card-title">Cloud Native</h3>
            <p className="card-text">
              Seamlessly integrated with Hugging Face LFS storage, providing a world-class CDN for all platform assets.
            </p>
          </div>

          <div className="feature-card">
            <div className="card-icon">🛡️</div>
            <h3 className="card-title">Proxy Logic</h3>
            <p className="card-text">
              Zero direct database access. Every query is routed through an intelligent HMAC-secured gateway.
            </p>
          </div>
        </section>

        {/* Independence Section */}
        <section className="indep-section">
          <div className="indep-info">
            <h2 className="indep-title">Totally Independent.<br/>Totally Autonomous.</h2>
            <p className="indep-text">
              Unlike traditional setups, BandhanNova platforms like Academy, Blogs, and Market do not rely on third-party backend services. 
              They rely on <strong>BFOBS</strong>—their own sovereign backend infrastructure.
            </p>
            <div style={{marginTop:'32px', color:'var(--primary)', fontWeight:700, fontSize:'14px', letterSpacing:'2px'}}>
              OWN THE INFRASTRUCTURE. OWN THE FUTURE.
            </div>
          </div>
          <div className="indep-visual"></div>
        </section>

        {/* Footer */}
        <footer style={{padding:'80px 0', borderTop:'1px solid var(--glass-border)', textAlign:'center', color:'var(--text-dim)', fontSize:'12px'}}>
          <div style={{marginBottom:'20px', fontWeight:800, color:'#fff'}}>BANDHANNOVA PLATFORMS</div>
          &copy; 2026 BANDHANNOVA INFRASTRUCTURE UNIT. ALL RIGHTS RESERVED.
        </footer>
      </div>
    </main>
  );
}
