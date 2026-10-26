// Health check del contenedor (web/Dockerfile y Coolify). No toca el API:
// si el backend se cae, el blog sigue sirviendo lo que tiene en cache y no
// hace falta reiniciarlo.
export function GET() {
  return new Response("ok");
}
