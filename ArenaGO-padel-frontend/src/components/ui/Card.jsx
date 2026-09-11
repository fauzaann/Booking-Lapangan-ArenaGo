export default function Card({ className = "", padded = true, hover = false, children, ...props }) {
  return (
    <div
      className={`bg-surface-raised border border-border rounded-lg ${padded ? "p-6" : ""} ${
        hover ? "transition-shadow duration-200 hover:shadow-raised" : ""
      } ${className}`}
      {...props}
    >
      {children}
    </div>
  );
}
