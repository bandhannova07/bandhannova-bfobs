"use client";

import { useState, useEffect } from "react";
import styles from "./page.module.css";
import { fetchAPI } from "../../../lib/api";
import Link from "next/link";

export default function APIKeysPage() {
  const [sections, setSections] = useState([]);
  const [cards, setCards] = useState([]);
  const [keys, setKeys] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedSection, setSelectedSection] = useState(null);
  const [selectedCard, setSelectedCard] = useState(null);
  
  // Modals
  const [showSectionModal, setShowSectionModal] = useState(false);
  const [showCardModal, setShowCardModal] = useState(false);
  const [showKeyModal, setShowKeyModal] = useState(false);
  const [showSecret, setShowSecret] = useState({});

  // Form Data
  const [sectionForm, setSectionForm] = useState({ name: "" });
  const [keyForm, setKeyForm] = useState({ label: "API Key", value: "", values: [], api_url: "", use_url: false });

  const handleSmartPaste = (e) => {
    const text = e.clipboardData.getData("text");
    const lines = text.split(/\r?\n/).filter(line => line.trim() !== "");
    
    if (lines.length > 1) {
      e.preventDefault();
      // Split into individual values for bulk mode
      setKeyForm(prev => ({ ...prev, values: lines }));
    }
  };

  const removeKeyFromBulk = (index) => {
    setKeyForm(prev => ({ ...prev, values: prev.values.filter((_, i) => i !== index) }));
  };

  const updateBulkKey = (index, newVal) => {
    const updated = [...keyForm.values];
    updated[index] = newVal;
    setKeyForm(prev => ({ ...prev, values: updated }));
  };

  const loadData = async () => {
    setLoading(true);
    try {
      const res = await fetchAPI("/admin/api/sections");
      if (res.success) setSections(res.sections || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const loadCards = async (sectionId) => {
    try {
      const res = await fetchAPI(`/admin/api/cards?section_id=${sectionId}`);
      if (res.success) setCards(res.cards || []);
    } catch (err) {
      console.error(err);
    }
  };

  const loadKeys = async (cardId) => {
    try {
      const res = await fetchAPI(`/admin/api/keys?card_id=${cardId}`);
      if (res.success) setKeys(res.keys || []);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleAddSection = async (e) => {
    e.preventDefault();
    const res = await fetchAPI("/admin/api/sections", { method: "POST", body: JSON.stringify(sectionForm) });
    if (res.success) {
      setShowSectionModal(false);
      setSectionForm({ name: "" });
      loadData();
    }
  };

  const [cardForm, setCardForm] = useState({ 
    name: "", 
    icon: "🔑", 
    endpoint_url: "", 
    platform_type: "openai_compatible", 
    limit_rps: 0, 
    limit_rpm: 0, 
    limit_rph: 0, 
    limit_rpd: 0, 
    limit_rpmonth: 0, 
    limit_concurrent: 0, 
    section_id: "" 
  });
  const [editingCard, setEditingCard] = useState(null);

  const handleAddCard = async (e) => {
    e.preventDefault();
    const url = editingCard ? `/admin/api/cards/${editingCard.id}` : "/admin/api/cards";
    const res = await fetchAPI(url, { 
      method: editingCard ? "PUT" : "POST", 
      body: JSON.stringify({ ...cardForm, section_id: selectedSection.id }) 
    });
    if (res.success) {
      setShowCardModal(false);
      setEditingCard(null);
      setCardForm({ 
        name: "", 
        icon: "🔑", 
        endpoint_url: "", 
        platform_type: "openai_compatible", 
        limit_rps: 0, 
        limit_rpm: 0, 
        limit_rph: 0, 
        limit_rpd: 0, 
        limit_rpmonth: 0, 
        limit_concurrent: 0, 
        section_id: "" 
      });
      loadCards(selectedSection.id);
    }
  };

  const openEditCard = (card) => {
    setEditingCard(card);
    setCardForm({ 
      name: card.name, 
      icon: card.icon, 
      endpoint_url: card.endpoint_url, 
      platform_type: card.platform_type, 
      limit_rps: card.limit_rps, 
      limit_rpm: card.limit_rpm, 
      limit_rph: card.limit_rph, 
      limit_rpd: card.limit_rpd, 
      limit_rpmonth: card.limit_rpmonth, 
      limit_concurrent: card.limit_concurrent, 
      section_id: card.section_id 
    });
    setShowCardModal(true);
  };

  const handleAddKey = async (e) => {
    e.preventDefault();
    try {
      const payload = {
        card_id: selectedCard.id,
        label: "API Key", // Default label
      };

      if (keyForm.values.length > 0) {
        payload.values = keyForm.values;
      } else {
        payload.value = keyForm.value;
      }

      await fetchAPI("/admin/api/keys", {
        method: "POST",
        body: JSON.stringify(payload)
      });
      setShowKeyModal(false);
      setKeyForm({ label: "API Key", value: "", values: [], api_url: "", use_url: false });
      loadKeys(selectedCard.id);
    } catch (err) {
      alert(err.message);
    }
  };

  const handleDelete = async (type, id) => {
    if (!confirm(`Move this ${type} to Unused APIs?`)) return;
    const res = await fetchAPI(`/admin/api/items/${type}/${id}/delete`, { method: "POST" });
    if (res.success) {
      if (type === "card") loadCards(selectedSection.id);
      if (type === "key") loadKeys(selectedCard.id);
    }
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div className={styles.headerLeft}>
          <h2 className={styles.title}>
            {selectedCard ? selectedCard.name : (selectedSection ? selectedSection.name : "API Command Center")}
          </h2>
          <div className={styles.breadcrumb}>
            <span onClick={() => { setSelectedSection(null); setSelectedCard(null); }}>Sections</span>
            {selectedSection && (
              <>
                <span className={styles.sep}>/</span>
                <span onClick={() => { setSelectedCard(null); loadCards(selectedSection.id); }}>{selectedSection.name}</span>
              </>
            )}
            {selectedCard && (
              <>
                <span className={styles.sep}>/</span>
                <span>{selectedCard.name}</span>
              </>
            )}
          </div>
        </div>

        <div className={styles.headerRight}>
          {!selectedSection && (
            <button className="btn btn-primary" onClick={() => setShowSectionModal(true)}>+ New Section</button>
          )}
          {selectedSection && !selectedCard && (
            <button className="btn btn-primary" onClick={() => setShowCardModal(true)}>+ Add API Card</button>
          )}
          {selectedCard && (
            <div className={styles.endpointBar}>
              <span className={styles.endpointLabel}>Endpoint:</span>
              <code className={styles.endpointCode}>
                /{selectedSection.name.toLowerCase().replace(/\(/g, '').replace(/\)/g, '').replace(/ /g, '-')}/{selectedCard.name.toLowerCase().replace(/\(/g, '').replace(/\)/g, '').replace(/ /g, '-')}/execute
              </code>
              <button 
                className={styles.copyBtn} 
                onClick={() => {
                  const url = `/${selectedSection.name.toLowerCase().replace(/\(/g, '').replace(/\)/g, '').replace(/ /g, '-')}/${selectedCard.name.toLowerCase().replace(/\(/g, '').replace(/\)/g, '').replace(/ /g, '-')}/execute`;
                  navigator.clipboard.writeText(url);
                  alert("Endpoint copied!");
                }}
              >
                Copy
              </button>
              <button className="btn btn-primary" onClick={() => setShowKeyModal(true)}>+ Add Key</button>
            </div>
          )}
        </div>
      </div>

      {loading ? (
        <div className={styles.loading}>Synchronizing API Registry...</div>
      ) : (
        <div className={styles.content}>
          {!selectedSection && (
            <div className={styles.grid}>
              {sections.map(s => (
                <div key={s.id} className={styles.card} onClick={() => { setSelectedSection(s); loadCards(s.id); }}>
                  <div className={styles.cardGlow}></div>
                  <div className={styles.cardIcon}>{s.id === 'unused' ? '♻️' : '📁'}</div>
                  <div className={styles.cardName}>{s.name}</div>
                  <div className={styles.cardMeta}>{s.cardCount} Cards</div>
                </div>
              ))}
            </div>
          )}

          {selectedSection && !selectedCard && (
            <div className={styles.grid}>
              {cards.map(c => (
                <div key={c.id} className={styles.card} onClick={() => { setSelectedCard(c); loadKeys(c.id); }}>
                  <div className={styles.cardGlow} style={{ background: "var(--neon-purple)" }}></div>
                  <div className={styles.cardIcon}>{c.icon || "🔑"}</div>
                  <div className={styles.cardName}>{c.name}</div>
                  <div className={styles.cardMeta}>
                    <span>{c.key_count} Keys</span>
                    <div className={styles.cardActions}>
                      <button className={styles.editBtn} onClick={(e) => { e.stopPropagation(); openEditCard(c); }}>Edit</button>
                      <button className={styles.delBtn} onClick={(e) => { e.stopPropagation(); handleDelete("card", c.id); }}>Delete</button>
                    </div>
                  </div>
                </div>
              ))}
              {cards.length === 0 && <div className={styles.empty}>No API cards in this section.</div>}
            </div>
          )}

          {selectedCard && (
            <div className={styles.keyList}>
              {keys.map(k => (
                <div key={k.id} className={styles.keyItem}>
                  <div className={styles.keyMain}>
                    <div className={styles.keyInfo}>
                      <div className={styles.keyLabel}>{k.label}</div>
                      <div className={styles.keyValue}>
                        <code>{showSecret[k.id] ? k.id : k.masked_value}</code>
                        <button onClick={() => setShowSecret(prev => ({ ...prev, [k.id]: !prev[k.id] }))}>
                          {showSecret[k.id] ? "Hide" : "Show"}
                        </button>
                      </div>
                    </div>
                    <div className={styles.keyActions}>
                      <span className={`badge ${k.status === 'active' ? 'badge-online' : 'badge-offline'}`}>{k.status}</span>
                      <button className={styles.delBtn} onClick={() => handleDelete("key", k.id)}>Delete</button>
                    </div>
                  </div>
                  
                  <div className={styles.keyUsage}>
                    {selectedCard.limit_rps > 0 && <span className={styles.usageTag}>SEC: {k.usage?.sec || 0}/{selectedCard.limit_rps}</span>}
                    {selectedCard.limit_rpm > 0 && <span className={styles.usageTag}>MIN: {k.usage?.min || 0}/{selectedCard.limit_rpm}</span>}
                    {selectedCard.limit_rph > 0 && <span className={styles.usageTag}>HOUR: {k.usage?.hour || 0}/{selectedCard.limit_rph}</span>}
                    {selectedCard.limit_rpd > 0 && <span className={styles.usageTag}>DAY: {k.usage?.day || 0}/{selectedCard.limit_rpd}</span>}
                    {selectedCard.limit_rpmonth > 0 && <span className={styles.usageTag}>MONTH: {k.usage?.month || 0}/{selectedCard.limit_rpmonth}</span>}
                  </div>
                </div>
              ))}
              {keys.length === 0 && <div className={styles.empty}>No keys added to this card yet.</div>}
            </div>
          )}
        </div>
      )}

      {/* Modals */}
      {showSectionModal && (
        <div className={styles.modalOverlay} onClick={(e) => e.target === e.currentTarget && setShowSectionModal(false)}>
          <div className={styles.modal}>
            <div className={styles.modalHeader}>
              <span>Initialize New API Section</span>
              <button className={styles.closeBtn} onClick={() => setShowSectionModal(false)}>×</button>
            </div>
            <form onSubmit={handleAddSection} style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
              <div className={styles.formGroup}>
                <label>Section Title (e.g. AI Engine APIs)</label>
                <input type="text" className={styles.input} value={sectionForm.name} onChange={e => setSectionForm({ name: e.target.value })} placeholder="Enter category name..." required />
              </div>
              <div className={styles.modalFooter}>
                <button type="button" className="btn btn-glass" onClick={() => setShowSectionModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Create Section</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showCardModal && (
        <div className={styles.modalOverlay} onClick={(e) => e.target === e.currentTarget && setShowCardModal(false)}>
          <div className={styles.modal}>
            <div className={styles.modalHeader}>
              <span>{editingCard ? "Modify API Provider" : "Register API Provider Card"}</span>
              <button className={styles.closeBtn} onClick={() => { setShowCardModal(false); setEditingCard(null); }}>×</button>
            </div>
            <form onSubmit={handleAddCard} style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
              <div className={styles.formGroup}>
                <label>Provider Name</label>
                <input type="text" className={styles.input} value={cardForm.name} onChange={e => setCardForm({ ...cardForm, name: e.target.value })} placeholder="e.g. OpenRouter" required />
              </div>
              <div className={styles.formGroup}>
                <label>Base Endpoint URL</label>
                <input type="text" className={styles.input} value={cardForm.endpoint_url} onChange={e => setCardForm({ ...cardForm, endpoint_url: e.target.value })} placeholder="https://api.example.com/v1" required />
              </div>
              <div className={styles.formGrid}>
                <div className={styles.formGroup}>
                  <label>Req Per Second (RPS)</label>
                  <input type="number" className={styles.input} value={cardForm.limit_rps} onChange={e => setCardForm({ ...cardForm, limit_rps: parseInt(e.target.value) || 0 })} placeholder="0" />
                </div>
                <div className={styles.formGroup}>
                  <label>Req Per Minute (RPM)</label>
                  <input type="number" className={styles.input} value={cardForm.limit_rpm} onChange={e => setCardForm({ ...cardForm, limit_rpm: parseInt(e.target.value) || 0 })} placeholder="0" />
                </div>
                <div className={styles.formGroup}>
                  <label>Req Per Hour (RPH)</label>
                  <input type="number" className={styles.input} value={cardForm.limit_rph} onChange={e => setCardForm({ ...cardForm, limit_rph: parseInt(e.target.value) || 0 })} placeholder="0" />
                </div>
                <div className={styles.formGroup}>
                  <label>Req Per Day (RPD)</label>
                  <input type="number" className={styles.input} value={cardForm.limit_rpd} onChange={e => setCardForm({ ...cardForm, limit_rpd: parseInt(e.target.value) || 0 })} placeholder="0" />
                </div>
                <div className={styles.formGroup}>
                  <label>Req Per Month</label>
                  <input type="number" className={styles.input} value={cardForm.limit_rpmonth} onChange={e => setCardForm({ ...cardForm, limit_rpmonth: parseInt(e.target.value) || 0 })} placeholder="0" />
                </div>
                <div className={styles.formGroup}>
                  <label>Concurrent Req</label>
                  <input type="number" className={styles.input} value={cardForm.limit_concurrent} onChange={e => setCardForm({ ...cardForm, limit_concurrent: parseInt(e.target.value) || 0 })} placeholder="0" />
                </div>
              </div>
              <div className={styles.hint}>* Set to 0 for unlimited requests.</div>
              <div className={styles.modalFooter}>
                <button type="button" className="btn btn-glass" onClick={() => { setShowCardModal(false); setEditingCard(null); }}>Cancel</button>
                <button type="submit" className="btn btn-primary">{editingCard ? "Save Changes" : "Add Provider"}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showKeyModal && (
        <div className={styles.modalOverlay} onClick={(e) => e.target === e.currentTarget && setShowKeyModal(false)}>
          <div className={styles.modal}>
            <div className={styles.modalHeader}>
              <div style={{ display: "flex", flexDirection: "column", gap: "4px" }}>
                <span>Secure API Key Management</span>
                <span style={{ fontSize: "10px", color: "var(--text-secondary)" }}>
                  Sensitive keys are encrypted using AES-256-GCM.
                </span>
              </div>
              <div style={{ display: "flex", gap: "8px" }}>
                <button className={styles.closeBtn} onClick={() => setShowKeyModal(false)}>×</button>
              </div>
            </div>
            <form onSubmit={handleAddKey}>
              <div className={styles.formGroup}>
                <label>API Key(s)</label>
                <p className={styles.hint}>Paste your keys below. Multiple lines will be auto-detected.</p>
                
                {keyForm.values.length > 0 ? (
                  <div className={styles.bulkContainer}>
                    {keyForm.values.map((val, idx) => (
                      <div key={idx} className={styles.keyBox}>
                        <div className={styles.keyBoxHeader}>
                          <span>Key Box #{idx + 1}</span>
                          <button type="button" className={styles.removeBtn} onClick={() => removeKeyFromBulk(idx)}>×</button>
                        </div>
                        <input 
                          type="password" 
                          className={styles.input} 
                          value={val} 
                          onChange={e => updateBulkKey(idx, e.target.value)} 
                        />
                      </div>
                    ))}
                    <button type="button" className="btn btn-glass" style={{ width: "100%", marginTop: "8px" }} onClick={() => setKeyForm(prev => ({ ...prev, values: [...prev.values, ""] }))}>+ Add Manual Box</button>
                  </div>
                ) : (
                  <textarea 
                    className={styles.input} 
                    style={{ minHeight: "150px", resize: "none" }}
                    value={keyForm.value} 
                    onChange={e => setKeyForm({ ...keyForm, value: e.target.value })} 
                    onPaste={handleSmartPaste}
                    placeholder="Paste single key or hundreds of keys here..." 
                    required={keyForm.values.length === 0}
                  />
                )}
              </div>

              <div className={styles.modalFooter} style={{ marginTop: "24px" }}>
                <button type="button" className="btn btn-glass" onClick={() => setShowKeyModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save All Keys</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
