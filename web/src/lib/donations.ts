// Links de donación configurables por entorno. Los placeholders apuntan a
// perfiles todavía no creados — al tener las cuentas reales alcanza con
// definir las env en Vercel / el pipeline, sin cambiar código.
const CAFECITO_URL = process.env.NEXT_PUBLIC_CAFECITO_URL ?? "https://cafecito.app/radar-global";
const TECITO_URL = process.env.NEXT_PUBLIC_TECITO_URL ?? "https://tecito.app/radar-global";

export { CAFECITO_URL, TECITO_URL };
