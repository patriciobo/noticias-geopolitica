import { useId } from "react";
import { REGION_THEME } from "@/lib/regionTheme";
import styles from "./RegionBanner.module.css";

export default function RegionBanner({ region }: { region: string }) {
  const theme = REGION_THEME[region];
  const gradientId = useId();

  if (!theme) {
    return <h3>{region}</h3>;
  }

  return (
    <svg
      viewBox="0 0 600 90"
      className={styles.banner}
      role="img"
      aria-label={`Región: ${region}`}
    >
      <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor={theme.from} />
          <stop offset="100%" stopColor={theme.to} />
        </linearGradient>
      </defs>
      <rect width="600" height="90" rx="14" fill={`url(#${gradientId})`} />
      <circle cx="555" cy="18" r="46" fill="white" opacity="0.08" />
      <circle cx="500" cy="78" r="28" fill="white" opacity="0.08" />
      <circle cx="590" cy="70" r="14" fill="white" opacity="0.08" />
      <text x="26" y="58" fontSize="34">
        {theme.emoji}
      </text>
      <text x="76" y="56" fontSize="26" fontWeight="700" fill="white">
        {region}
      </text>
    </svg>
  );
}
