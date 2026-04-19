# Hugging Face Deployment Guide for BandhanNova API Hunter

Bhai, Hugging Face Spaces (Docker) e deploy kora khub e easy. Ekhane setup korle tumi **16GB RAM** r **CPU** prochur free pabe, jeta onno kothao pabe na.

### 1. New Space Create Koro
- [huggingface.co/new-space](https://huggingface.co/new-space) e jao.
- **Space Name:** `api-hunter` (ba jekono name).
- **SDK:** Choice koro **Docker**.
- **Template:** Select koro **Blank**.
- **Public/Private:** Tumi initially Private o rakhte paro jodi chao.

### 2. Secrets Add Koro (IMPORTANT)
Space create hobar por **Settings** tab e jao, tarpor **Variables and secrets** section e click koro. Ekhane amader `.env` file er sob key gulo "Secret" hisebe add korte hobe:
- `BANDHANNOVA_MASTER_KEY`: (Tomar bnova_secret_gateway_key_2026)
- `OPENROUTER_KEYS`: (Comma separated keys)
- `TAVILY_KEYS`: (Comma separated keys)
- `PORT`: **7860** (Hugging Face default port)

### 3. Code Upload Koro
Tumi direct GitHub connect korte paro ba Hugging Face e ekti repo create kore ekhane files gulo push korte paro. 
Files gulo hobe:
- `main.go`
- `Dockerfile`
- `go.mod`
- `go.sum`
- `config/`, `handlers/`, `middleware/`, `proxy/` folders.

### 4. Final Endpoint
Deploy hoye gele, tomar final URL hobe:
`https://<username>-<space-name>.hf.space/v1/ai/chat`

---

### Verification
Deploy hobar por, tumi Smartpedia-r `.env.local` e amader notun gateway URL ta diye dilei puro ecosystem connected hoye jabe! ✨
