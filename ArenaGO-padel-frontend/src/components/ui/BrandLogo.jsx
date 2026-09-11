export default function BrandLogo({ compact = false }) {
  return (
    <span className="flex items-center gap-2.5">
      <span className="flex h-9 w-9 items-center justify-center rounded-full bg-onyx text-[11px] font-bold tracking-[-0.04em] text-canvas">
        AG
      </span>
      {!compact && <span className="font-display text-lg leading-none">ArenaGO</span>}
    </span>
  );
}
