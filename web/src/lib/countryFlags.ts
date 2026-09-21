// Mismos nombres de país que usa config/sources.yaml en el backend.
export const COUNTRY_FLAGS: Record<string, string> = {
  Canadá: "🇨🇦",
  "Estados Unidos": "🇺🇸",
  México: "🇲🇽",
  Colombia: "🇨🇴",
  Venezuela: "🇻🇪",
  Brasil: "🇧🇷",
  Perú: "🇵🇪",
  Chile: "🇨🇱",
  Argentina: "🇦🇷",
  China: "🇨🇳",
  Japón: "🇯🇵",
  "Corea del Sur": "🇰🇷",
  Rusia: "🇷🇺",
  "Reino Unido": "🇬🇧",
  Alemania: "🇩🇪",
  España: "🇪🇸",
  Italia: "🇮🇹",
  Francia: "🇫🇷",
};

export function flagFor(country: string): string {
  return COUNTRY_FLAGS[country] ?? "🏳️";
}
