import { useQuery } from "@tanstack/react-query";
import { useState, useMemo, type FC } from "react";
import classNames from "classnames";
import { CheckCircle, AlertTriangle, XCircle, Clock } from "lucide-react";
import { ErrorFallback } from "../../../components/helpers/ErrorFallback";
import { serviceService } from "../../../services/services";
import { calculateUptimeForLogs } from "../../../utils/uptime";

type Props = {
  service_id: string;
};

export const UptimeSection: FC<Props> = ({ service_id }) => {
  const [rangeDays, setRangeDays] = useState<7 | 30>(7);

  const operationsQuery = useQuery({
    queryKey: ["operation-logs", service_id],
    queryFn: () => serviceService.fetchOperationLogs(service_id),
    refetchInterval: 5000,
  });

  const uptimeStats = useMemo(() => {
    return calculateUptimeForLogs(operationsQuery.data ?? []);
  }, [operationsQuery.data]);

  if (operationsQuery.isLoading) return <div className="p-4">Cargando métricas de uptime...</div>;
  if (operationsQuery.isError) {
    return (
      <ErrorFallback
        error={operationsQuery.error}
        onRetry={() => setTimeout(operationsQuery.refetch)}
      />
    );
  }

  const selectedStat = rangeDays === 7 ? uptimeStats.last7d : uptimeStats.last30d;

  const getStatusColor = (percent: number) => {
    if (percent >= 99) return "text-success border-success/30 bg-success/5";
    if (percent >= 95) return "text-warning border-warning/30 bg-warning/5";
    return "text-error border-error/30 bg-error/5";
  };

  const getStatusIcon = (percent: number) => {
    if (percent >= 99) return <CheckCircle className="w-8 h-8 text-success" />;
    if (percent >= 95) return <AlertTriangle className="w-8 h-8 text-warning" />;
    return <XCircle className="w-8 h-8 text-error" />;
  };

  return (
    <div className="space-y-6">
      {/* Selector de Rango */}
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-bold uppercase tracking-wider text-base-content/60">
          Porcentaje de disponibilidad
        </h4>
        <div className="flex gap-1">
          {([7, 30] as const).map(d => (
            <button
              key={d}
              className={classNames("btn btn-xs", {
                "btn-neutral": rangeDays === d,
                "btn-ghost": rangeDays !== d,
              })}
              onClick={() => setRangeDays(d)}
            >
              Últimos {d} días
            </button>
          ))}
        </div>
      </div>

      {/* Tarjeta Principal de Disponibilidad */}
      <div className={classNames("card border shadow-sm p-6 flex flex-col md:flex-row items-center gap-6", getStatusColor(selectedStat.percent))}>
        <div className="shrink-0">{getStatusIcon(selectedStat.percent)}</div>
        <div className="flex-grow text-center md:text-left">
          <div className="text-4xl font-black text-base-content leading-none">
            {selectedStat.percent.toFixed(2)}%
          </div>
          <p className="text-xs text-base-content/65 mt-1 font-medium">
            Disponibilidad promedio en los últimos {rangeDays} días.
          </p>
        </div>
        <div className="text-xs font-mono text-base-content/85 grid grid-cols-2 gap-x-4 gap-y-1 bg-base-200/50 p-3 rounded-lg border border-base-200/40">
          <div>Tiempo Activo:</div>
          <div className="font-bold text-success text-right">
            {Math.round(selectedStat.activeMs / 3600000)}h
          </div>
          <div>Tiempo Inactivo:</div>
          <div className="font-bold text-error text-right">
            {Math.round(selectedStat.inactiveMs / 3600000)}h
          </div>
        </div>
      </div>

      {/* Historial de períodos */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="card border border-base-200 bg-base-100/50 p-4">
          <span className="text-[10px] uppercase font-bold text-base-content/40">Últimas 24 Horas</span>
          <span className="text-2xl font-extrabold text-base-content mt-1">
            {uptimeStats.last24h.percent.toFixed(2)}%
          </span>
        </div>
        <div className="card border border-base-200 bg-base-100/50 p-4">
          <span className="text-[10px] uppercase font-bold text-base-content/40">Últimos 7 Días</span>
          <span className="text-2xl font-extrabold text-base-content mt-1">
            {uptimeStats.last7d.percent.toFixed(2)}%
          </span>
        </div>
        <div className="card border border-base-200 bg-base-100/50 p-4">
          <span className="text-[10px] uppercase font-bold text-base-content/40">Últimos 30 Días</span>
          <span className="text-2xl font-extrabold text-base-content mt-1">
            {uptimeStats.last30d.percent.toFixed(2)}%
          </span>
        </div>
      </div>

      {/* Resumen explicativo */}
      <div className="text-xs text-base-content/60 flex items-start gap-2 bg-base-200/30 p-3 rounded-lg border border-base-300/40">
        <Clock className="w-4 h-4 shrink-0 mt-0.5" />
        <p>
          El cálculo de uptime se basa en el registro de eventos exitosos de la instancia. Las operaciones de tipo
          <span className="font-bold text-primary mx-1">start</span> y <span className="font-bold text-primary mr-1">wakeup</span> indican estado encendido, mientras que las operaciones <span className="font-bold text-primary mx-1">stop</span> y <span className="font-bold text-primary mr-1">sleep</span> representan estado apagado.
        </p>
      </div>
    </div>
  );
};
