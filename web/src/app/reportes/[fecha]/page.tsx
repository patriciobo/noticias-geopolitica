import { notFound } from "next/navigation";
import { fetchReportByDate, ReportNotFoundError } from "@/lib/api";
import ReportView from "@/components/ReportView";

export default async function ReportePage({
  params,
}: PageProps<"/reportes/[fecha]">) {
  const { fecha } = await params;

  try {
    const report = await fetchReportByDate(fecha);
    return <ReportView date={report.date} markdown={report.markdown} sources={report.sources} />;
  } catch (err) {
    if (err instanceof ReportNotFoundError) {
      notFound();
    }
    throw err;
  }
}
