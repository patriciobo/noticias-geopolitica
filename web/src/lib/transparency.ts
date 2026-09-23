// Repositorio público del proyecto: todo lo que se afirma en /metodologia y
// en el bloque "Cómo se hizo esta edición" se puede verificar ahí.
export const REPO_URL = process.env.REPO_URL ?? "https://github.com/patriciobo/noticias-geopolitica";

// "dueño/repo", para el comando de verificación de atestaciones.
export const REPO_SLUG = REPO_URL.replace(/^https:\/\/github\.com\//, "").replace(/\/$/, "");

export function repoFileURL(path: string, ref = "main"): string {
  return `${REPO_URL}/blob/${ref}/${path}`;
}

// El repo completo tal como estaba en un commit dado.
export function repoTreeURL(ref: string): string {
  return `${REPO_URL}/tree/${ref}`;
}

export function repoHistoryURL(path: string): string {
  return `${REPO_URL}/commits/main/${path}`;
}
