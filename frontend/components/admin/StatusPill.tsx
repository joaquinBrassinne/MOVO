import { STATUS_LABEL } from "@/lib/admin/messages";
import { BUSINESS_STATUS, type BusinessStatus } from "@/lib/business/types";

const TONE: Record<BusinessStatus, string> = {
  [BUSINESS_STATUS.DRAFT]: "bg-movo-sand text-movo-slate",
  [BUSINESS_STATUS.PENDING_REVIEW]: "bg-movo-accent/40 text-movo-ink",
  [BUSINESS_STATUS.ACTIVE]: "bg-movo-ok/15 text-movo-ok",
  [BUSINESS_STATUS.PAUSED]: "bg-movo-slate/20 text-movo-ink",
  [BUSINESS_STATUS.DELETED]: "bg-movo-danger/15 text-movo-danger",
};

/** La etiqueta de color del estado de un local. */
export function StatusPill({ status }: { status: BusinessStatus }) {
  return (
    <span className={`inline-flex rounded-full px-3 py-1 text-xs font-bold ${TONE[status]}`}>
      {STATUS_LABEL[status]}
    </span>
  );
}
