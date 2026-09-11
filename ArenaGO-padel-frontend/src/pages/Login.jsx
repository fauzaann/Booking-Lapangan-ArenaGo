import { useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import Input from "../components/ui/Input";
import Button from "../components/ui/Button";
import { useAuth } from "../context/AuthContext";

export default function Login() {
  const { login, status } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const redirectTo = location.state?.from?.pathname || "/";

  const [form, setForm] = useState({ email: "", password: "" });
  const [error, setError] = useState("");

  function update(field, value) {
    setForm((prev) => ({ ...prev, [field]: value }));
  }

  async function handleSubmit(e) {
    e.preventDefault();
    if (!form.email || !form.password) {
      setError("Email dan password wajib diisi.");
      return;
    }
    setError("");
    try {
      await login(form);
      navigate(redirectTo, { replace: true });
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <div className="mx-auto flex min-h-[calc(100vh-72px)] max-w-md flex-col justify-center px-5 py-12 sm:px-0">
      <div className="mb-8 text-center">
        <span className="mx-auto mb-4 flex h-11 w-11 items-center justify-center rounded-full bg-onyx text-canvas">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
            <path
              d="M12 2C7 2 4 6 4 11c0 4 2 6 4 8 1.2 1.2 2.6 2 4 2s2.8-.8 4-2c2-2 4-4 4-8 0-5-3-9-8-9Z"
              stroke="currentColor"
              strokeWidth="1.6"
            />
            <path d="M12 6.5v11" stroke="currentColor" strokeWidth="1.6" />
          </svg>
        </span>
        <h1 className="mb-2 font-display text-3xl">Masuk ke Akun Anda</h1>
        <p className="text-sm text-ink-muted">
          Kelola booking dan riwayat main padel Anda di satu tempat.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label="Email"
          type="email"
          placeholder="nama@email.com"
          value={form.email}
          onChange={(e) => update("email", e.target.value)}
        />
        <Input
          label="Password"
          type="password"
          placeholder="••••••••"
          value={form.password}
          onChange={(e) => update("password", e.target.value)}
          error={error}
        />

        <div className="flex items-center justify-between text-sm">
          <label className="flex items-center gap-2 text-ink-muted">
            <input type="checkbox" className="h-4 w-4 rounded border-border" />
            Ingat saya
          </label>
          <button type="button" className="text-ink-muted hover:text-ink hover:underline">
            Lupa password?
          </button>
        </div>

        <Button type="submit" className="w-full" disabled={status === "loading"}>
          {status === "loading" ? "Memproses..." : "Masuk"}
        </Button>
      </form>

      <p className="mt-8 text-center text-sm text-ink-muted">
        Belum punya akun?{" "}
        <Link to="/register" className="font-medium text-ink hover:underline">
          Daftar sekarang
        </Link>
      </p>
    </div>
  );
}
