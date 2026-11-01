import { useEffect, useMemo, useState } from "react";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import Icon from "../components/ui/Icon";
import Input from "../components/ui/Input";
import { api } from "../lib/api";
import { formatIDR } from "../lib/format";
import {
  FIELD_DAYS,
  createDefaultSchedules,
  toFieldForm,
  toFieldPayload,
  validateFieldForm,
} from "../lib/adminFields";

const STATUS_LABELS = {
  ACTIVE: "Aktif",
  INACTIVE: "Nonaktif",
  MAINTENANCE: "Dalam perawatan",
};

export default function AdminFields() {
  const { id } = useParams();
  return id ? <AdminFieldForm id={id} /> : <AdminFieldList />;
}

function AdminFieldList() {
  const location = useLocation();
  const [fields, setFields] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    api.adminFields()
      .then((result) => {
        if (active) setFields(result.data || []);
      })
      .catch((requestError) => {
        if (active) setError(requestError.message);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <Link to="/admin" className="mb-3 inline-flex items-center gap-1 text-sm text-ink-muted hover:text-ink">
            <Icon name="arrow_back" size={16} />
            Dashboard admin
          </Link>
          <p className="mb-2 text-xs font-bold uppercase tracking-[0.12em] text-champagne">ArenaGO Admin</p>
          <h1 className="font-display text-3xl sm:text-4xl">Kelola lapangan</h1>
          <p className="mt-2 text-ink-muted">Atur informasi, harga, gambar, dan jam operasional lapangan.</p>
        </div>
        <Link to="/admin/lapangan/baru">
          <Button icon={<Icon name="add" size={18} />}>Tambah lapangan</Button>
        </Link>
      </div>

      {location.state?.message && <p role="status" className="mb-5 rounded-md bg-sage/10 p-4 text-sm text-sage">{location.state.message}</p>}
      {error && <p role="alert" className="mb-5 rounded-md bg-terracotta/10 p-4 text-sm text-terracotta">{error}</p>}
      {loading ? (
        <p className="py-16 text-center text-sm text-ink-muted">Memuat daftar lapangan...</p>
      ) : fields.length === 0 ? (
        <Card className="py-14 text-center">
          <Icon name="sports_tennis" size={36} className="text-ink-faint" />
          <p className="mt-3 font-display text-xl">Belum ada lapangan</p>
          <p className="mt-1 text-sm text-ink-muted">Tambahkan lapangan pertama agar dapat dikelola di sini.</p>
        </Card>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          {fields.map((field) => (
            <Card key={field.id} className="flex flex-col gap-4 sm:flex-row">
              <div className="relative h-40 shrink-0 overflow-hidden rounded-md bg-sage/10 sm:w-48">
                {field.image_url ? (
                  <img
                    src={field.image_url}
                    alt={`Gambar ${field.name}`}
                    className="h-full w-full object-cover"
                    onError={(event) => { event.currentTarget.style.display = "none"; }}
                  />
                ) : (
                  <div className="flex h-full items-center justify-center text-sage">
                    <Icon name="sports_tennis" size={36} />
                  </div>
                )}
              </div>
              <div className="flex min-w-0 flex-1 flex-col">
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div className="min-w-0">
                    <h2 className="truncate font-display text-xl">{field.name}</h2>
                    <p className="mt-1 text-sm text-ink-muted">{field.location}</p>
                  </div>
                  <span className={`rounded-full px-3 py-1 text-xs font-medium ${field.status === "ACTIVE" ? "bg-sage/10 text-sage" : "bg-champagne/30 text-ink-muted"}`}>
                    {STATUS_LABELS[field.status] || field.status}
                  </span>
                </div>
                <p className="mt-3 line-clamp-2 text-sm text-ink-muted">{field.description || "Belum ada deskripsi."}</p>
                <p className="mt-3 text-sm font-semibold">{formatIDR(field.price_per_hour)} <span className="font-normal text-ink-muted">/ jam</span></p>
                <Link to={`/admin/lapangan/${field.id}`} className="mt-4 self-start">
                  <Button variant="secondary" size="sm" icon={<Icon name="edit" size={16} />}>Edit lapangan</Button>
                </Link>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

function AdminFieldForm({ id }) {
  const navigate = useNavigate();
  const isNew = id === "baru";
  const [form, setForm] = useState(() => ({
    ...toFieldForm(),
    schedules: createDefaultSchedules(),
  }));
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [loadError, setLoadError] = useState("");
  const [fieldErrors, setFieldErrors] = useState({});
  const [originalStatus, setOriginalStatus] = useState("");
  const [originalSchedules, setOriginalSchedules] = useState([]);

  useEffect(() => {
    if (isNew) return undefined;
    let active = true;
    Promise.all([api.field(id), api.fieldSchedules(id)])
      .then(([fieldsResult, schedulesResult]) => {
        const field = fieldsResult.data;
        if (!field) throw new Error("Lapangan tidak ditemukan.");
        if (active) {
          const fieldForm = toFieldForm(field, schedulesResult.data || []);
          setForm(fieldForm);
          setOriginalStatus(field.status);
          setOriginalSchedules(fieldForm.schedules);
        }
      })
      .catch((requestError) => {
        if (active) {
          setError(requestError.message);
          setLoadError(requestError.message);
        }
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [id, isNew]);

  const previewURL = useMemo(() => {
    try {
      const url = new URL(form.image_url.trim());
      return ["http:", "https:"].includes(url.protocol) ? url.href : "";
    } catch {
      return "";
    }
  }, [form.image_url]);

  function updateForm(key, value) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  function updateSchedule(day, key, value) {
    setForm((current) => ({
      ...current,
      schedules: current.schedules.map((schedule) =>
        schedule.day === day ? { ...schedule, [key]: value } : schedule,
      ),
    }));
  }

  async function handleSubmit(event) {
    event.preventDefault();
    const errors = validateFieldForm(form);
    setFieldErrors(errors);
    if (Object.keys(errors).length) {
      setError("Periksa kembali data lapangan dan jam operasional.");
      return;
    }
    const changedSchedule = !isNew && JSON.stringify(form.schedules) !== JSON.stringify(originalSchedules);
    const deactivating = !isNew && originalStatus === "ACTIVE" && form.status !== "ACTIVE";
    const warnings = [];
    if (changedSchedule) warnings.push("Perubahan jam akan mengubah ketersediaan booking berikutnya.");
    if (deactivating) warnings.push("Lapangan akan berhenti menerima booking baru.");
    if (warnings.length && !window.confirm(`${warnings.join(" ")} Booking yang sudah ada tidak akan dihapus. Lanjutkan?`)) {
      return;
    }

    setSaving(true);
    setError("");
    try {
      const payload = toFieldPayload(form);
      if (isNew) await api.createField(payload);
      else await api.updateField(id, payload);
      navigate("/admin/lapangan", {
        state: { message: isNew ? "Lapangan berhasil ditambahkan." : "Perubahan lapangan berhasil disimpan." },
      });
    } catch (requestError) {
      setError(requestError.message);
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <p className="px-5 py-20 text-center text-sm text-ink-muted">Memuat data lapangan...</p>;
  if (loadError) {
    return (
      <div className="px-5 py-20 text-center">
        <p role="alert" className="text-sm text-terracotta">{loadError}</p>
        <Link to="/admin/lapangan" className="mt-4 inline-block text-sm font-medium text-sage hover:underline">Kembali ke daftar lapangan</Link>
      </div>
    );
  }

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <div className="mx-auto max-w-4xl">
        <Link to="/admin/lapangan" className="mb-5 inline-flex items-center gap-1 text-sm text-ink-muted hover:text-ink">
          <Icon name="arrow_back" size={16} />
          Kembali ke daftar lapangan
        </Link>
        <div className="mb-6">
          <p className="mb-2 text-xs font-bold uppercase tracking-[0.12em] text-champagne">ArenaGO Admin</p>
          <h1 className="font-display text-3xl sm:text-4xl">{isNew ? "Tambah lapangan" : "Edit lapangan"}</h1>
          <p className="mt-2 text-sm text-ink-muted">Lengkapi detail dan jam operasional. Perubahan jadwal tidak menghapus booking yang sudah ada.</p>
        </div>

        {error && <p role="alert" className="mb-5 rounded-md bg-terracotta/10 p-4 text-sm text-terracotta">{error}</p>}

        <form onSubmit={handleSubmit} className="space-y-6">
          <Card>
            <h2 className="mb-5 font-display text-xl">Informasi lapangan</h2>
            <div className="grid gap-5 sm:grid-cols-2">
              <Input
                label="Nama lapangan"
                required
                minLength={3}
                maxLength={120}
                value={form.name}
                error={fieldErrors.name}
                onChange={(event) => updateForm("name", event.target.value)}
              />
              <Input
                label="Lokasi"
                required
                minLength={3}
                maxLength={200}
                value={form.location}
                error={fieldErrors.location}
                onChange={(event) => updateForm("location", event.target.value)}
              />
              <Input
                label="Harga per jam (Rp)"
                type="number"
                min="1"
                step="1000"
                required
                value={form.price_per_hour}
                error={fieldErrors.price_per_hour}
                onChange={(event) => updateForm("price_per_hour", event.target.value)}
              />
              <label className="block text-xs font-medium tracking-wide text-ink-muted">
                Status
                <select
                  value={form.status}
                  onChange={(event) => updateForm("status", event.target.value)}
                  className="mt-1.5 h-[50px] w-full rounded-md border border-border bg-surface-raised px-4 text-[15px] text-ink focus:border-onyx focus:outline-none focus:ring-4 focus:ring-onyx/5"
                >
                  <option value="ACTIVE">Aktif — menerima booking</option>
                  <option value="INACTIVE">Nonaktif</option>
                  <option value="MAINTENANCE">Dalam perawatan</option>
                </select>
              </label>
              <label className="block text-xs font-medium tracking-wide text-ink-muted sm:col-span-2">
                Deskripsi
                <textarea
                  rows={4}
                  maxLength={1000}
                  value={form.description}
                  onChange={(event) => updateForm("description", event.target.value)}
                  className={`mt-1.5 w-full rounded-md border bg-surface-raised px-4 py-3 text-[15px] text-ink focus:outline-none focus:ring-4 ${fieldErrors.description ? "border-terracotta focus:ring-terracotta/10" : "border-border focus:border-onyx focus:ring-onyx/5"}`}
                />
                <span className={`mt-1 block text-right text-xs ${fieldErrors.description ? "text-terracotta" : "text-ink-faint"}`}>
                  {fieldErrors.description || `${form.description.length}/1000`}
                </span>
              </label>
              <Input
                label="Fasilitas"
                hint="Pisahkan beberapa fasilitas dengan koma."
                value={form.facilities}
                error={fieldErrors.facilities}
                onChange={(event) => updateForm("facilities", event.target.value)}
              />
              <Input
                label="URL gambar"
                type="url"
                maxLength={500}
                placeholder="https://contoh.com/lapangan.jpg"
                hint="Masukkan tautan gambar publik yang diawali http:// atau https://."
                value={form.image_url}
                error={fieldErrors.image_url}
                onChange={(event) => updateForm("image_url", event.target.value)}
              />
              <div className="sm:col-span-2">
                <p className="mb-2 text-xs font-medium tracking-wide text-ink-muted">Pratinjau gambar</p>
                <div className="flex h-48 items-center justify-center overflow-hidden rounded-md border border-border bg-sage/10">
                  {previewURL ? (
                    <img src={previewURL} alt="Pratinjau lapangan" className="h-full w-full object-cover" />
                  ) : (
                    <div className="text-center text-ink-faint">
                      <Icon name="image" size={36} />
                      <p className="mt-1 text-xs">Masukkan URL gambar untuk melihat pratinjau.</p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </Card>

          <Card>
            <div className="mb-5">
              <h2 className="font-display text-xl">Jam operasional mingguan</h2>
              <p className="mt-1 text-sm text-ink-muted">Hari baru default tutup. Hari yang ditandai tutup tidak akan menampilkan slot booking.</p>
            </div>
            {fieldErrors.schedules && <p role="alert" className="mb-4 text-sm text-terracotta">{fieldErrors.schedules}</p>}
            <div className="space-y-3">
              {FIELD_DAYS.map(({ value: day, label }) => {
                const schedule = form.schedules.find((item) => item.day === day);
                return (
                  <div key={day} className="grid gap-3 rounded-md border border-border p-3 sm:grid-cols-[1fr_auto_1fr_1fr] sm:items-end">
                    <p className="pb-2 text-sm font-medium">{label}</p>
                    <label className="flex items-center gap-2 pb-3 text-sm text-ink-muted">
                      <input
                        type="checkbox"
                        checked={schedule.is_closed}
                        onChange={(event) => updateSchedule(day, "is_closed", event.target.checked)}
                        className="h-4 w-4 accent-sage"
                      />
                      Tutup
                    </label>
                    <Input
                      label="Buka"
                      type="time"
                      value={schedule.open_time}
                      disabled={schedule.is_closed}
                      onChange={(event) => updateSchedule(day, "open_time", event.target.value)}
                    />
                    <Input
                      label="Tutup"
                      type="time"
                      value={schedule.close_time}
                      disabled={schedule.is_closed}
                      onChange={(event) => updateSchedule(day, "close_time", event.target.value)}
                    />
                  </div>
                );
              })}
            </div>
          </Card>

          <div className="flex flex-wrap justify-end gap-3">
            <Link to="/admin/lapangan">
              <Button type="button" variant="secondary" disabled={saving}>Batal</Button>
            </Link>
            <Button type="submit" disabled={saving} icon={<Icon name="save" size={18} />}>
              {saving ? "Menyimpan..." : "Simpan lapangan"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
