import { notFound } from "next/navigation";
import { fetchReportByDate, ReportNotFoundError, type ReportResponse } from "@/lib/api";
import ReportView from "@/components/ReportView";
import DonationSupport from "@/components/DonationSupport";

// Ver REVALIDATE_SECONDS en lib/api.ts (acá tiene que ser un literal).
export const revalidate = 300;

// Lista vacía: no se prerenderiza nada en el build, pero cada fecha se
// renderiza la primera vez que alguien la pide y queda cacheada (ISR) —
// sin esto la ruta se renderiza en cada request.
export async function generateStaticParams() {
  return [];
}

// La fecha se interpola en la URL de la API: validarla antes evita que un
// parámetro armado (ej. "..%2Fsubscribers") haga que el fetch server-side
// termine en otro endpoint del backend.
const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;

export default async function ReportePage({
  params,
}: PageProps<"/reportes/[fecha]">) {
  const { fecha } = await params;
  if (!DATE_RE.test(fecha)) {
    notFound();
  }

  let report: ReportResponse;
  try {
    report = await fetchReportByDate(fecha);
  } catch (err) {
    if (err instanceof ReportNotFoundError) {
      notFound();
    }
    throw err;
  }

  return (
    <>
      <ReportView date={report.date} markdown={report.markdown} sources={report.sources} provenance={report.provenance} />
      <DonationSupport variant="article" />
    </>
  );
}
