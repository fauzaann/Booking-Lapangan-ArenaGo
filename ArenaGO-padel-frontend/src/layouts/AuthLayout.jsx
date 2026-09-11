import { Link, Outlet } from "react-router-dom";
import BrandLogo from "../components/ui/BrandLogo";

export default function AuthLayout() {
  return (
    <div className="min-h-screen bg-canvas">
      <header className="border-b border-border px-5 py-5 sm:px-8">
        <Link to="/" className="inline-flex items-center gap-2.5">
          <BrandLogo />
        </Link>
      </header>
      <Outlet />
    </div>
  );
}
