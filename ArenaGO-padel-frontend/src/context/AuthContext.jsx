import { createContext, useContext, useEffect, useState } from "react";
import { api, clearSession, saveSession } from "../lib/api";

const STORAGE_KEY = "arenago:session";

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      return raw ? JSON.parse(raw) : null;
    } catch {
      return null;
    }
  });
  const [status, setStatus] = useState("idle"); // idle | loading

  useEffect(() => {
    if (user) localStorage.setItem(STORAGE_KEY, JSON.stringify(user));
    else localStorage.removeItem(STORAGE_KEY);
  }, [user]);

  async function login(credentials) {
    setStatus("loading");
    try {
      const result = await api.login(credentials);
      saveSession(result.data);
      setUser(result.data.user);
      return result.data.user;
    } finally {
      setStatus("idle");
    }
  }

  async function register(credentials) {
    setStatus("loading");
    try {
      const result = await api.register(credentials);
      saveSession(result.data);
      setUser(result.data.user);
      return result.data.user;
    } finally {
      setStatus("idle");
    }
  }

  function updateProfile(partial) {
    setUser((prev) => (prev ? { ...prev, ...partial } : prev));
  }

  function logout() {
    clearSession();
    setUser(null);
  }

  const value = {
    user,
    isAuthenticated: !!user,
    isAdmin: user?.role === "admin",
    status,
    login,
    register,
    logout,
    updateProfile,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
