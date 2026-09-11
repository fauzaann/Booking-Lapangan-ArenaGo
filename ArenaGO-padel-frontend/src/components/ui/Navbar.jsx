import { Link, NavLink, useNavigate } from "react-router-dom";
import { useState } from "react";
import Icon from "./Icon";
import { useAuth } from "../../context/AuthContext";
import BrandLogo from "./BrandLogo";

function getNavItems({ isAuthenticated, isAdmin }) {
  const items = [{ to: "/", label: "Eksplor", icon: "sports_tennis" }];
  if (isAuthenticated) {
    items.push({ to: "/riwayat", label: "Riwayat", icon: "receipt_long" });
  }
  if (isAdmin) {
    items.push({ to: "/admin/transaksi", label: "Transaksi", icon: "account_balance_wallet" });
    items.push({ to: "/admin", label: "Kelola Jadwal", icon: "calendar_month" });
  }
  return items;
}

export default function Navbar() {
  const { user, isAuthenticated, isAdmin, logout } = useAuth();
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);
  const navItems = getNavItems({ isAuthenticated, isAdmin });

  function handleLogout() {
    setMenuOpen(false);
    logout();
    navigate("/");
  }

  return (
    <header className="glass sticky top-0 z-50 border-b border-border">
      <div className="mx-auto flex h-18 max-w-[1440px] items-center justify-between px-5 sm:px-8 lg:px-16">
        <Link to="/" className="flex items-center gap-2.5">
          <BrandLogo />
        </Link>

        <nav className="hidden items-center gap-1 md:flex">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === "/" || item.to === "/admin"}
              className={({ isActive }) =>
                `rounded-full px-4 py-2 text-sm font-medium transition-colors duration-150 ${
                  isActive ? "bg-onyx text-canvas" : "text-ink-muted hover:text-ink"
                }`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>

        <div className="flex items-center gap-3">
          {!isAuthenticated ? (
            <div className="flex items-center gap-2">
              <Link
                to="/login"
                className="hidden text-sm font-medium text-ink-muted hover:text-ink sm:block"
              >
                Masuk
              </Link>
              <Link
                to="/register"
                className="rounded-full bg-onyx px-4 py-2 text-sm font-medium text-canvas hover:bg-onyx-hover"
              >
                Daftar
              </Link>
            </div>
          ) : (
            <div className="relative">
              <button
                onClick={() => setMenuOpen((v) => !v)}
                className="flex items-center gap-2 rounded-full border border-border py-1 pl-1 pr-3 hover:border-ink-muted"
              >
                <span className="flex h-8 w-8 items-center justify-center rounded-full bg-gradient-to-br from-champagne to-onyx text-xs font-semibold text-canvas">
                  {user.name.split(" ").map((n) => n[0]).slice(0, 2).join("")}
                </span>
                <span className="hidden text-sm font-medium sm:block">{user.name.split(" ")[0]}</span>
                <Icon name="expand_more" size={16} className="text-ink-faint" />
              </button>

              {menuOpen && (
                <>
                  <div className="fixed inset-0 z-10" onClick={() => setMenuOpen(false)} />
                  <div className="absolute right-0 z-20 mt-2 w-52 overflow-hidden rounded-md border border-border bg-surface-raised shadow-overlay">
                    <Link
                      to="/profil"
                      onClick={() => setMenuOpen(false)}
                      className="flex items-center gap-2.5 px-4 py-3 text-sm hover:bg-surface"
                    >
                      <Icon name="person" size={18} />
                      Profil Saya
                    </Link>
                    <Link
                      to="/riwayat"
                      onClick={() => setMenuOpen(false)}
                      className="flex items-center gap-2.5 px-4 py-3 text-sm hover:bg-surface md:hidden"
                    >
                      <Icon name="receipt_long" size={18} />
                      Riwayat
                    </Link>
                    <button
                      onClick={handleLogout}
                      className="flex w-full items-center gap-2.5 border-t border-border px-4 py-3 text-left text-sm text-terracotta hover:bg-terracotta/5"
                    >
                      <Icon name="logout" size={18} />
                      Keluar
                    </button>
                  </div>
                </>
              )}
            </div>
          )}
        </div>
      </div>
    </header>
  );
}

export { getNavItems };
