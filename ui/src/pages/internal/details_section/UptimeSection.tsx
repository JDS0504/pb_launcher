import { useQuery } from "@tanstack/react-query";
import { useMemo, type FC } from "react";
import classNames from "classnames";
import { Clock } from "lucide-react";
import { ErrorFallback } from "../../../components/helpers/ErrorFallback";
import { serviceService } from "../../../services/services";
import { calculateUptimeForLogs } from "../../../utils/uptime";

type Props = {
  service_id: string;
};

export const UptimeSection: FC<Props> = ({ service_id }) => {
  const operationsQuery = useQuery({
    queryKey: ["operation-logs", service_id],
    queryFn: () => serviceService.fetchOperationLogs(service_id),
    refetchInterval: 5000,
  });

  const uptimeStats = useMemo(() => {
    return calculateUptimeForLogs(operationsQuery.data ?? []);
  }, [operationsQuery.data]);

  if (operationsQuery.isLoading) return <div className="p-4 text-xs text-base-content/60">Cargando métricas de uptime...</div>;
  if (operationsQuery.isError) {
    return (
      <ErrorFallback
        error={operationsQuery.error}
        onRetry={() => setTimeout(operationsQuery.refetch)}
      />
    );
  }

  const getStatusColor = (percent: number) => {
    if (percent >= 99) return "text-success border-success/30 bg-success/5";
    if (percent >= 95) return "text-warning border-warning/30 bg-warning/5";
    return "text-error border-error/30 bg-error/5";
  };

  const getRadialColorClass = (percent: number) => {
    if (percent >= 99) return "text-success";
    if (percent >= 95) return "text-warning";
    return "text-error";
  };

  const renderCard = (title: string, stats: { percent: number; activeMs: number; inactiveMs: number }) => {
    const cardBgColor = getStatusColor(stats.percent);
    const progressColor = getRadialColorClass(stats.percent);
    return (
      <div className={classNames("card border shadow-sm p-4 flex flex-col items-center justify-between text-center gap-4 transition-all hover:shadow-md", cardBgColor)}>
        <span className="text-[10px] uppercase font-bold tracking-wider text-base-content/60">{title}</span>
        
        <div 
          className={classNames("radial-progress shrink-0", progressColor)}
          style={{
            "--value": Math.round(stats.percent),
            "--size": "5.5rem",
            "--thickness": "8px",
          } as React.CSSProperties}
          role="progressbar"
        >
          <span className="text-xs md:text-sm font-extrabold text-base-content">{stats.percent.toFixed(1)}%</span>
        </div>

        <div className="text-[10px] font-mono w-full bg-base-200/50 p-2 rounded border border-base-200/30 leading-relaxed text-base-content/75">
          <div className="flex justify-between border-b border-base-200/30 pb-0.5">
            <span>Tiempo Activo:</span>
            <span className="font-bold text-success">{(stats.activeMs / 3600000).toFixed(1)}h</span>
          </div>
          <div className="flex justify-between pt-0.5">
            <span>Tiempo Inactivo:</span>
            <span className="font-bold text-error">{(stats.inactiveMs / 3600000).toFixed(1)}h</span>
          </div>
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-bold uppercase tracking-wider text-base-content/60">
          Porcentaje de disponibilidad de la instancia
        </h4>
      </div>

      {/* Grid de 3 Tarjetas de Disponibilidad */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {renderCard("Últimas 24 Horas", uptimeStats.last24h)}
        {renderCard("Últimos 7 Días", uptimeStats.last7d)}
        {renderCard("Últimos 30 Días", uptimeStats.last30d)}
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
