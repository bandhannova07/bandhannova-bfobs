import { API_URL } from "./constants";

// Helper to get token (only runs on client side in this architecture)
const getToken = () => {
    if (typeof window !== "undefined") {
        // Since we are not using cookies right now, we store token in memory or localStorage cache
        // Note: Implementation plan suggested purely memory, but for a next.js app where pages might refresh,
        // session storage is a slightly safer middle ground that survives refresh but dies when tab closes.
        return sessionStorage.getItem("admin_token");
    }
    return null;
};

export const setToken = (token) => {
    if (typeof window !== "undefined") {
        sessionStorage.setItem("admin_token", token);
    }
};

export const clearToken = () => {
    if (typeof window !== "undefined") {
        sessionStorage.removeItem("admin_token");
    }
};

export const fetchAPI = async (endpoint, options = {}) => {
    const token = getToken();
    
    const defaultHeaders = {
        "Content-Type": "application/json",
    };

    if (token) {
        defaultHeaders["X-Admin-Token"] = token;
    }

    const config = {
        ...options,
        headers: {
            ...defaultHeaders,
            ...options.headers,
        },
    };

    const res = await fetch(`${API_URL}${endpoint}`, config);

    let data;
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
