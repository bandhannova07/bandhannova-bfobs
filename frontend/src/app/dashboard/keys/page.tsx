"use client";

import React, { useState, useEffect } from "react";
import styles from "./page.module.css";
import { fetchAPI } from "../../../lib/api";

interface Usage {
  sec: number;
  min: number;
  hour: number;
  day: number;
  month: number;
}

interface Key {
  id: string;
  label: string;
  masked_value: string;
  status: string;
  usage?: Usage;
}

interface Card {
  id: string;
  name: string;
  icon: string;
  key_count: number;
  endpoint_url: string;
  platform_type: string;
  limit_rps: number;
  limit_rpm: number;
  limit_rph: number;
  limit_rpd: number;
  limit_rpmonth: number;
  limit_concurrent: number;
  section_id: string;
}

interface Section {
  id: string;
  name: string;
  cardCount: number;
}

interface KeyForm {
  label: string;
  value: string;
  values: string[];
  api_url: string;
  use_url: boolean;
}

interface CardForm {
  name: string;
  icon: string;
  endpoint_url: string;
  platform_type: string;
  limit_rps: number;
  limit_rpm: number;
  limit_rph: number;
  limit_rpd: number;
  limit_rpmonth: number;
  limit_concurrent: number;
  section_id: string;
}

export default function APIKeysPage() {
  const [sections, setSections] = useState<Section[]>([]);
  const [cards, setCards] = useState<Card[]>([]);
  const [keys, setKeys] = useState<Key[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedSection, setSelectedSection] = useState<Section | null>(null);
  const [selectedCard, setSelectedCard] = useState<Card | null>(null);
  
  // Modals
  const [showSectionModal, setShowSectionModal] = useState(false);
  const [showCardModal, setShowCardModal] = useState(false);
  const [showKeyModal, setShowKeyModal] = useState(false);
  const [showSecret, setShowSecret] = useState<Record<string, boolean>>({});

  // Form Data
  const [sectionForm, setSectionForm] = useState({ name: "" });
  const [keyForm, setKeyForm] = useState<KeyForm>({ label: "API Key", value: "", values: [], api_url: "", use_url: false });

  const [cardForm, setCardForm] = useState<CardForm>({ 
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
  const [editingCard, setEditingCard] = useState<Card | null>(null);

  const handleSmartPaste = (e: React.ClipboardEvent) => {
    const text = e.clipboardData.getData("text");
    const lines = text.split(/\r?\n/).filter(line => line.trim() !== "");
    
    if (lines.length > 1) {
      e.preventDefault();
      setKeyForm(prev => ({ ...prev, values: lines }));
    }
  };

  const removeKeyFromBulk = (index: number) => {
    setKeyForm(prev => ({ ...prev, values: prev.values.filter((_, i) => i !== index) }));
  };

  const updateBulkKey = (index: number, newVal: string) => {
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

  const loadCards = async (sectionId: string) => {
    try {
      const res = await fetchAPI(`/admin/api/cards?section_id=${sectionId}`);
      if (res.success) setCards(res.cards || []);
    } catch (err) {
      console.error(err);
    }
  };

  const loadKeys = async (cardId: string) => {
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

  const handleAddSection = async (e: React.FormEvent) => {
    e.preventDefault();
    const res = await fetchAPI("/admin/api/sections", { method: "POST", body: JSON.stringify(sectionForm) });
    if (res.success) {
      setShowSectionModal(false);
      setSectionForm({ name: "" });
      loadData();
    }
  };

  const handleAddCard = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedSection) return;
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

  const openEditCard = (card: Card) => {
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

  const handleAddKey = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedCard) return;
    try {
      const payload: any = {
        card_id: selectedCard.id,
        label: keyForm.label || "API Key",
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
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleDelete = async (type: "card" | "key", id: string) => {
    if (!confirm(`Move this ${type} to archive?`)) return;
    const res = await fetchAPI(`/admin/api/items/${type}/${id}/delete`, { method: "POST" });
    if (res.success) {
      if (type === "card" && selectedSection) loadCards(selectedSection.id);
      if (type === "key" && selectedCard) loadKeys(selectedCard.id);
    }
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div className={styles.headerLeft}>
          <h2 className={styles.title}>
            {selectedCard ? selectedCard.name : (selectedSection ? selectedSection.name : "Registry Control")}
          </h2>
          <div className={styles.breadcrumb}>
            <span onClick={() => { setSelectedSection(null); setSelectedCard(null); }}>REGISTRY</span>
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
            <button className="btn btn-primary" onClick={() => setShowCardModal(true)}>+ New API Card</button>
          )}
          {selectedCard && selectedSection && (
            <div className={styles.endpointBar}>
              <span className={styles.endpointLabel}>ENDPOINT</span>
              <code className={styles.endpointCode}>
                /{selectedSection.name.toLowerCase().replace(/[^a-z0-9]/g, '-')}/{selectedCard.name.toLowerCase().replace(/[^a-z0-9]/g, '-')}/execute
              </code>
              <button 
                className={styles.copyBtn} 
                onClick={() => {
                  const url = `/${selectedSection!.name.toLowerCase().replace(/[^a-z0-9]/g, '-')}/${selectedCard!.name.toLowerCase().replace(/[^a-z0-9]/g, '-')}/execute`;
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
        <div className={styles.loading}>SYNCING REGISTRY...</div>
      ) : (
        <div className={styles.content}>
          {!selectedSection && (
            <div className={styles.grid}>
              {sections.map(s => (
                <div key={s.id} className={`glass-panel ${styles.card}`} onClick={() => { setSelectedSection(s); loadCards(s.id); }}>
                  <div className={styles.cardIcon}>{s.id === 'unused' ? '♻️' : '📁'}</div>
                  <div className={styles.cardName}>{s.name}</div>
                  <div className={styles.cardMeta}>{s.cardCount} CARDS ACTIVE</div>
                </div>
              ))}
            </div>
          )}

          {selectedSection && !selectedCard && (
            <div className={styles.grid}>
              {cards.map(c => (
                <div key={c.id} className={`glass-panel ${styles.card}`} onClick={() => { setSelectedCard(c); loadKeys(c.id); }}>
                  <div className={styles.cardIcon}>{c.icon || "🔑"}</div>
                  <div className={styles.cardName}>{c.name}</div>
                  <div className={styles.cardMeta}>
                    <span>{c.key_count} KEYS</span>
                    <div className={styles.cardActions}>
                      <button className={styles.editBtn} onClick={(e) => { e.stopPropagation(); openEditCard(c); }}>Edit</button>
                      <button className={styles.delBtn} onClick={(e) => { e.stopPropagation(); handleDelete("card", c.id); }}>Del</button>
                    </div>
                  </div>
                </div>
              ))}
              {cards.length === 0 && <div className={styles.empty}>Section is empty.</div>}
            </div>
          )}

          {selectedCard && (
            <div className={styles.keyList}>
              {keys.map(k => (
                <div key={k.id} className={`glass-panel ${styles.keyItem}`}>
                  <div className={styles.keyMain}>
                    <div className={styles.keyInfo}>
                      <div className={styles.keyLabel}>{k.label}</div>
                      <div className={styles.keyValue}>
                        <code>{showSecret[k.id] ? k.id : k.masked_value}</code>
                        <button onClick={() => setShowSecret(prev => ({ ...prev, [k.id]: !prev[k.id] }))}>
                          {showSecret[k.id] ? "HIDE" : "SHOW"}
                        </button>
                      </div>
                    </div>
                    <div className={styles.keyActions}>
                      <span className={`badge ${k.status === 'active' ? 'badge-online' : 'badge-offline'}`}>{k.status}</span>
                      <button className={styles.delBtn} onClick={() => handleDelete("key", k.id)}>Archive</button>
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
              {keys.length === 0 && <div className={styles.empty}>No active keys found.</div>}
            </div>
          )}
        </div>
      )}

      {/* Modals */}
      {showSectionModal && (
        <div className={styles.modalOverlay} onClick={(e) => e.target === e.currentTarget && setShowSectionModal(false)}>
          <div className={`glass-panel ${styles.modal}`}>
            <div className={styles.modalHeader}>
              <span>Initialize Section</span>
              <button className={styles.closeBtn} onClick={() => setShowSectionModal(false)}>×</button>
            </div>
            <form onSubmit={handleAddSection} style={{ display: "flex", flexDirection: "column", gap: "24px" }}>
              <input type="text" className={styles.input} value={sectionForm.name} onChange={e => setSectionForm({ name: e.target.value })} placeholder="Section Category Name..." required autoFocus />
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
          <div className={`glass-panel ${styles.modal}`}>
            <div className={styles.modalHeader}>
              <span>{editingCard ? "Modify Provider" : "Register Provider"}</span>
              <button className={styles.closeBtn} onClick={() => { setShowCardModal(false); setEditingCard(null); }}>×</button>
            </div>
            <form onSubmit={handleAddCard} style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
              <input type="text" className={styles.input} value={cardForm.name} onChange={e => setCardForm({ ...cardForm, name: e.target.value })} placeholder="Provider Name (e.g. OpenAI)" required />
              <input type="text" className={styles.input} value={cardForm.endpoint_url} onChange={e => setCardForm({ ...cardForm, endpoint_url: e.target.value })} placeholder="Endpoint URL..." required />
              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "12px" }}>
                <input type="number" className={styles.input} value={cardForm.limit_rps} onChange={e => setCardForm({ ...cardForm, limit_rps: parseInt(e.target.value) || 0 })} placeholder="RPS Limit" />
                <input type="number" className={styles.input} value={cardForm.limit_rpm} onChange={e => setCardForm({ ...cardForm, limit_rpm: parseInt(e.target.value) || 0 })} placeholder="RPM Limit" />
              </div>
              <div className={styles.modalFooter}>
                <button type="button" className="btn btn-glass" onClick={() => { setShowCardModal(false); setEditingCard(null); }}>Cancel</button>
                <button type="submit" className="btn btn-primary">{editingCard ? "Update" : "Register"}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showKeyModal && (
        <div className={styles.modalOverlay} onClick={(e) => e.target === e.currentTarget && setShowKeyModal(false)}>
          <div className={`glass-panel ${styles.modal}`}>
            <div className={styles.modalHeader}>
              <span>Key Management</span>
              <button className={styles.closeBtn} onClick={() => setShowKeyModal(false)}>×</button>
            </div>
            <form onSubmit={handleAddKey}>
              {keyForm.values.length > 0 ? (
                <div className={styles.bulkContainer}>
                  {keyForm.values.map((val, idx) => (
                    <div key={idx} className={styles.keyBox}>
                      <div className={styles.keyBoxHeader}>
                        <span>KEY #{idx + 1}</span>
                        <button type="button" className={styles.removeBtn} onClick={() => removeKeyFromBulk(idx)}>×</button>
                      </div>
                      <input type="password" className={styles.input} value={val} onChange={e => updateBulkKey(idx, e.target.value)} />
                    </div>
                  ))}
                  <button type="button" className="btn btn-glass" style={{ width: "100%", marginTop: "8px" }} onClick={() => setKeyForm(prev => ({ ...prev, values: [...prev.values, ""] }))}>+ Add Box</button>
                </div>
              ) : (
                <textarea 
                  className={styles.input} 
                  style={{ minHeight: "150px", resize: "none" }}
                  value={keyForm.value} 
                  onChange={e => setKeyForm({ ...keyForm, value: e.target.value })} 
                  onPaste={handleSmartPaste}
                  placeholder="Paste single or multiple keys..." 
                  required={keyForm.values.length === 0}
                />
              )}
              <div className={styles.modalFooter} style={{ marginTop: "24px" }}>
                <button type="button" className="btn btn-glass" onClick={() => setShowKeyModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Secure Save</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

