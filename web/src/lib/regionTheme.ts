// Ids internos de región (config/sources.yaml) a su nombre legible — mismo
// mapeo que regionLabels en core/internal/report/synthesize.go, en el mismo
// orden editorial.
export const REGION_LABELS: Record<string, string> = {
  north_america: "América del Norte",
  latin_america: "América Latina",
  europe: "Europa",
  east_asia: "Asia Oriental",
  eurasia: "Eurasia",
  africa: "África",
  oceania: "Oceanía",
};

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
  África: { emoji: "🌍" },
  Oceanía: { emoji: "🌏" },
};
