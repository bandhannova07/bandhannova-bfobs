import { API_URL } from "./constants";

// Helper to get token (only runs on client side in this architecture)
const getToken = (): string | null => {
    if (typeof window !== "undefined") {
        return sessionStorage.getItem("admin_token");
    }
    return null;
};

export const setToken = (token: string): void => {
    if (typeof window !== "undefined") {
        sessionStorage.setItem("admin_token", token);
    }
};

export const clearToken = (): void => {
    if (typeof window !== "undefined") {
        sessionStorage.removeItem("admin_token");
    }
};

export const fetchAPI = async (endpoint: string, options: RequestInit = {}): Promise<any> => {
    const token = getToken();
    
    const defaultHeaders: Record<string, string> = {
        "Content-Type": "application/json",
    };

    if (token) {
        defaultHeaders["X-Admin-Token"] = token;
    }

    const config: RequestInit = {
        ...options,
        headers: {
            ...defaultHeaders,
            ...options.headers as Record<string, string>,
        },
    };

    const res = await fetch(`${API_URL}${endpoint}`, config);

    let data: any;
    try {
        const text = await res.text();
        data = text ? JSON.parse(text) : {};
    } catch (err) {
        throw new Error(`API Endpoint not found or returned invalid response (Status: ${res.status}). Ensure the Go backend is running and URL is correct.`);
    }

    if (!res.ok) {
        if (res.status === 401 || res.status === 403) {
            clearToken();
            if (typeof window !== "undefined" && window.location.pathname !== "/") {
                window.location.href = "/";
            }
        }
        throw new Error(data.message || "API request failed");
    }

    return data;
};

// Products
export const getProducts = () => fetchAPI("/admin/products");
export const addProduct = (data: any) => fetchAPI("/admin/products", { method: "POST", body: JSON.stringify(data) });
export const updateProduct = (id: string, data: any) => fetchAPI(`/admin/products/${id}`, { method: "PUT", body: JSON.stringify(data) });
export const deleteProduct = (id: string, masterKey: string, confirmation: string) => 
    fetchAPI(`/admin/products/${id}/delete`, { 
        method: "POST", 
        body: JSON.stringify({ master_key: masterKey, confirmation }) 
    });

// Infrastructure (Default Shards)
export const getShards = () => fetchAPI("/admin/infrastructure/shards");
export const addShard = (data: any) => fetchAPI("/admin/infrastructure/shards", { method: "POST", body: JSON.stringify(data) });
export const updateShard = (id: string, data: any) => fetchAPI(`/admin/infrastructure/shards/${id}`, { method: "PUT", body: JSON.stringify(data) });
export const removeShard = (id: string) => fetchAPI(`/admin/infrastructure/shards/${id}`, { method: "DELETE" });
export const queryShard = (id: string, query: string) => fetchAPI(`/admin/infrastructure/shards/${id}/query`, { method: "POST", body: JSON.stringify({ query }) });
export const clearShard = (id: string) => fetchAPI(`/admin/infrastructure/shards/${id}/clear`, { method: "POST" });
export const initShard = (id: string) => fetchAPI(`/admin/infrastructure/shards/${id}/init`, { method: "POST" });

// Databases
export const getDatabases = () => fetchAPI("/admin/databases");
export const addDatabase = (data: any) => fetchAPI("/admin/databases", { method: "POST", body: JSON.stringify(data) });
export const getDatabaseDetails = (slug: string) => fetchAPI(`/admin/databases/${slug}`);

// Audit
export const getAuditLogs = () => fetchAPI("/admin/audit");
