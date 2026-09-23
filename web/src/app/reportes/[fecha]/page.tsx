import { notFound } from "next/navigation";
import { fetchReportByDate, ReportNotFoundError, type ReportResponse } from "@/lib/api";
import ReportView from "@/components/ReportView";
import DonationSupport from "@/components/DonationSupport";

export default async function ReportePage({
  params,
}: PageProps<"/reportes/[fecha]">) {
  const { fecha } = await params;

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
      <ReportView date={report.date} markdown={report.markdown} sources={report.sources} />
      <DonationSupport variant="article" />
    </>
  );
}
