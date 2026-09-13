export default function BrandLogo({ compact = false }) {
  return (
    <span className={`flex items-center ${compact ? "h-8 w-8" : "h-10 w-[9rem]"}`}>
      <img
        src="/logo.svg"
        alt="ArenaGO"
        className="h-full w-full object-contain"
      />
    </span>
  );
}
