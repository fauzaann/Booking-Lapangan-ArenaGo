import { NavLink } from "react-router-dom";
import Icon from "./Icon";
import { getNavItems } from "./Navbar";
import { useAuth } from "../../context/AuthContext";

export default function BottomNav() {
  const { isAuthenticated, isAdmin } = useAuth();
  const items = getNavItems({ isAuthenticated, isAdmin }).slice(0, 4);

  return (
    <nav className="glass fixed inset-x-0 bottom-0 z-50 border-t border-border pb-[var(--spacing-safe)] md:hidden">
      <div className="flex items-center justify-around px-2 py-2">
        {items.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === "/" || item.to === "/admin"}
            className={({ isActive }) =>
              `flex min-w-[64px] flex-col items-center gap-0.5 rounded-lg px-2 py-1.5 text-[11px] font-medium ${
                isActive ? "text-onyx" : "text-ink-faint"
              }`
            }
          >
            {({ isActive }) => (
              <>
                <Icon name={item.icon} filled={isActive} />
                {item.label}
              </>
            )}
          </NavLink>
        ))}
      </div>
    </nav>
  );
}
