import { createContext, useContext, useEffect, useState } from "react";
import { mockUsers } from "../lib/mockData";

const STORAGE_KEY = "atelier-padel:session";

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

  // NOTE: these are mock, frontend-only implementations. Replace the body
  // of each function with a real call to your RBAC backend — the shape of
  // `user` (id, name, email, phone, role) is kept generic on purpose.
  function login({ role = "customer" } = {}) {
    return new Promise((resolve) => {
      setStatus("loading");
      setTimeout(() => {
        setUser(mockUsers[role] ?? mockUsers.customer);
        setStatus("idle");
        resolve(mockUsers[role] ?? mockUsers.customer);
      }, 700);
    });
  }

  function register({ name, email, phone }) {
    return new Promise((resolve) => {
      setStatus("loading");
      setTimeout(() => {
        const newUser = {
          id: `usr-${Math.floor(Math.random() * 9000 + 1000)}`,
          name,
          email,
          phone,
          role: "customer",
          joinedAt: new Date().toISOString().slice(0, 10),
        };
        setUser(newUser);
        setStatus("idle");
        resolve(newUser);
      }, 700);
    });
  }

  function updateProfile(partial) {
    setUser((prev) => (prev ? { ...prev, ...partial } : prev));
  }

  function logout() {
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
