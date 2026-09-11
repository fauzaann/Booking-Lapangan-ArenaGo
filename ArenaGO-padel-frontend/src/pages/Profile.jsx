import { useState } from "react";
import { useNavigate } from "react-router-dom";
import Icon from "../components/ui/Icon";
import Card from "../components/ui/Card";
import Input from "../components/ui/Input";
import Button from "../components/ui/Button";
import StatusBadge from "../components/ui/StatusBadge";
import { useAuth } from "../context/AuthContext";

export default function Profile() {
  const { user, updateProfile, logout } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({ name: user.name, phone: user.phone });
  const [saved, setSaved] = useState(false);

  function update(field, value) {
    setForm((prev) => ({ ...prev, [field]: value }));
    setSaved(false);
  }

  function handleSave(e) {
    e.preventDefault();
    updateProfile(form);
    setSaved(true);
  }

  function handleLogout() {
    logout();
    navigate("/");
  }

  return (
    <div className="mx-auto max-w-2xl px-5 py-10 sm:px-8">
      <div className="mb-8 flex items-center gap-4">
        <div className="flex h-16 w-16 items-center justify-center rounded-full bg-gradient-to-br from-champagne to-onyx text-lg font-semibold text-canvas">
          {user.name.split(" ").map((n) => n[0]).slice(0, 2).join("")}
        </div>
        <div>
          <div className="mb-1 flex items-center gap-2">
            <h1 className="font-display text-2xl">{user.name}</h1>
            <StatusBadge
              status={user.role === "admin" ? "confirmed" : "available"}
              label={user.role === "admin" ? "Admin" : "Pelanggan"}
            />
          </div>
          <p className="text-sm text-ink-faint">Bergabung sejak {user.joinedAt}</p>
        </div>
      </div>

      <Card>
        <h2 className="mb-4 font-display text-xl">Informasi Akun</h2>
        <form onSubmit={handleSave} className="space-y-4">
          <Input label="Nama Lengkap" value={form.name} onChange={(e) => update("name", e.target.value)} />
          <Input label="Email" value={user.email} disabled className="opacity-60" />
          <Input
            label="Nomor WhatsApp"
            value={form.phone}
            onChange={(e) => update("phone", e.target.value)}
          />
          <div className="flex items-center gap-3 pt-2">
            <Button type="submit">Simpan Perubahan</Button>
            {saved && (
              <span className="flex items-center gap-1.5 text-sm text-sage">
                <Icon name="check_circle" size={16} filled />
                Tersimpan
              </span>
            )}
          </div>
        </form>
      </Card>

      <Card className="mt-6 border-terracotta/30">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="font-display text-lg">Keluar Akun</h2>
            <p className="text-sm text-ink-muted">Anda perlu login kembali untuk memesan lapangan.</p>
          </div>
          <Button variant="danger" onClick={handleLogout}>
            Logout
          </Button>
        </div>
      </Card>
    </div>
  );
}
