export const PROVIDERS = {
  OpenRouter: { icon: "🤖", color: "var(--neon-purple)", label: "AI Inference" },
  Groq: { icon: "🎙️", color: "var(--neon-green)", label: "Speech-to-Text" },
  Tavily: { icon: "🔍", color: "var(--neon-amber)", label: "Search Engine" },
  Resend: { icon: "📧", color: "var(--neon-blue)", label: "Email Delivery" },
  Cerebras: { icon: "🧠", color: "var(--neon-red)", label: "Ultra-Fast AI" },
  TwelveData: { icon: "📈", color: "#22c55e", label: "Market Data" },
};

export const API_URL = process.env.NEXT_PUBLIC_API_URL || "";
