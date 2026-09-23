export type RegionTheme = {
  emoji: string;
};

// Mismos nombres de región que emite el backend (ver regionLabels en
// core/internal/report/synthesize.go) — si no hay match, se muestra sin emoji.
export const REGION_THEME: Record<string, RegionTheme> = {
  "América del Norte": { emoji: "🗽" },
  "América Latina": { emoji: "🌎" },
  Europa: { emoji: "🏛️" },
  "Asia Oriental": { emoji: "🏯" },
  Eurasia: { emoji: "🌍" },
};
