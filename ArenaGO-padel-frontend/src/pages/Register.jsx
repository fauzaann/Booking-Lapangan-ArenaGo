import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import Input from "../components/ui/Input";
import Button from "../components/ui/Button";
import { useAuth } from "../context/AuthContext";

export default function Register() {
  const { register, status } = useAuth();
  const navigate = useNavigate();

  const [form, setForm] = useState({ name: "", email: "", phone: "", password: "" });
  const [errors, setErrors] = useState({});

  function update(field, value) {
    setForm((prev) => ({ ...prev, [field]: value }));
  }

  function validate() {
    const next = {};
    if (!form.name.trim()) next.name = "Nama wajib diisi.";
    if (!/^\S+@\S+\.\S+$/.test(form.email)) next.email = "Format email tidak valid.";
    if (form.phone.replace(/\D/g, "").length < 9) next.phone = "Nomor WhatsApp tidak valid.";
    if (form.password.length < 8) next.password = "Password minimal 8 karakter.";
    setErrors(next);
    return Object.keys(next).length === 0;
  }

  async function handleSubmit(e) {
    e.preventDefault();
    if (!validate()) return;
    try {
      await register(form);
      navigate("/", { replace: true });
    } catch (err) {
      setErrors({ form: err.message });
    }
  }

  return (
    <div className="mx-auto flex min-h-[calc(100vh-72px)] max-w-md flex-col justify-center px-5 py-12 sm:px-0">
      <div className="mb-8 text-center">
        <h1 className="mb-2 font-display text-3xl">Buat Akun Baru</h1>
        <p className="text-sm text-ink-muted">
          Daftar untuk mulai booking lapangan dan menyimpan riwayat main Anda.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        {errors.form && <p className="text-sm text-red-600">{errors.form}</p>}
        <Input
          label="Nama Lengkap"
          placeholder="Nama Anda"
          value={form.name}
          onChange={(e) => update("name", e.target.value)}
          error={errors.name}
        />
        <Input
          label="Email"
          type="email"
          placeholder="nama@email.com"
          value={form.email}
          onChange={(e) => update("email", e.target.value)}
          error={errors.email}
        />
        <Input
          label="Nomor WhatsApp"
          placeholder="+62 8xx-xxxx-xxxx"
          value={form.phone}
          onChange={(e) => update("phone", e.target.value)}
          error={errors.phone}
        />
        <Input
          label="Password"
          type="password"
          placeholder="Minimal 8 karakter"
          value={form.password}
          onChange={(e) => update("password", e.target.value)}
          error={errors.password}
        />

        <Button type="submit" className="w-full" disabled={status === "loading"}>
          {status === "loading" ? "Memproses..." : "Daftar"}
        </Button>
      </form>

      <p className="mt-8 text-center text-sm text-ink-muted">
        Sudah punya akun?{" "}
        <Link to="/login" className="font-medium text-ink hover:underline">
          Masuk
        </Link>
      </p>
    </div>
  );
}
