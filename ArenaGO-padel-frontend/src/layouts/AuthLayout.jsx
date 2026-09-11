import { Link, Outlet } from "react-router-dom";

export default function AuthLayout() {
  return (
    <div className="min-h-screen bg-canvas">
      <header className="border-b border-border px-5 py-5 sm:px-8">
        <Link to="/" className="inline-flex items-center gap-2.5">
          <span className="flex h-9 w-9 items-center justify-center rounded-full bg-onyx text-canvas">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none">
              <path
                d="M12 2C7 2 4 6 4 11c0 4 2 6 4 8 1.2 1.2 2.6 2 4 2s2.8-.8 4-2c2-2 4-4 4-8 0-5-3-9-8-9Z"
                stroke="currentColor"
                strokeWidth="1.6"
              />
              <path d="M12 6.5v11" stroke="currentColor" strokeWidth="1.6" />
            </svg>
          </span>
          <span className="font-display text-lg leading-none">Atelier Padel</span>
        </Link>
      </header>
      <Outlet />
    </div>
  );
}
