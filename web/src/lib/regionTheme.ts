export type RegionTheme = {
  emoji: string;
  from: string;
  to: string;
};

// Mismos nombres de región que emite el backend (ver regionLabels en
// core/internal/report/synthesize.go) — si no hay match, no se dibuja banner.
export const REGION_THEME: Record<string, RegionTheme> = {
  "América del Norte": { emoji: "🗽", from: "#1e3a8a", to: "#3b82f6" },
  "América Latina": { emoji: "🌎", from: "#14532d", to: "#22c55e" },
  Europa: { emoji: "🏛️", from: "#7c2d12", to: "#f59e0b" },
  "Asia Oriental": { emoji: "🏯", from: "#831843", to: "#ec4899" },
  Eurasia: { emoji: "🌍", from: "#312e81", to: "#818cf8" },
};
