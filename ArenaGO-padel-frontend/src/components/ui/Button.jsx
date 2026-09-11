const VARIANTS = {
  primary:
    "bg-onyx text-canvas hover:bg-onyx-hover active:scale-[0.99] disabled:bg-slate disabled:text-canvas/70",
  secondary:
    "bg-transparent text-onyx border border-champagne hover:bg-champagne-tint disabled:opacity-40",
  ghost:
    "bg-transparent text-onyx hover:underline underline-offset-4 px-3 disabled:opacity-40",
  danger:
    "bg-transparent text-terracotta border border-terracotta/40 hover:bg-terracotta/5",
};

const SIZES = {
  md: "h-12 px-6 text-[15px]",
  sm: "h-10 px-4 text-sm",
  lg: "h-14 px-8 text-base",
};

export default function Button({
  as: Tag = "button",
  variant = "primary",
  size = "md",
  className = "",
  icon,
  iconPosition = "left",
  children,
  ...props
}) {
  const shape = variant === "ghost" ? "" : "rounded-full";
  return (
    <Tag
      className={`inline-flex items-center justify-center gap-2 font-medium transition-all duration-150 ease-out disabled:cursor-not-allowed whitespace-nowrap ${shape} ${VARIANTS[variant]} ${SIZES[size]} ${className}`}
      {...props}
    >
      {icon && iconPosition === "left" && icon}
      {children}
      {icon && iconPosition === "right" && icon}
    </Tag>
  );
}
