import { useQuery } from "@tanstack/react-query";
import { useMemo, type FC } from "react";
import classNames from "classnames";
import { Clock } from "lucide-react";
import { ErrorFallback } from "../../../components/helpers/ErrorFallback";
import { serviceService } from "../../../services/services";

type Props = {
  service_id: string;
  serviceCreated?: string;
};

export const UptimeSection: FC<Props> = ({ service_id }) => {
  // Consultar la métrica de Uptime de la instancia específica en lugar de toda la lista
  const uptimeQuery = useQuery({
    queryKey: ["service-uptime", service_id],
    queryFn: () => serviceService.fetchServiceUptimeByID(service_id),
    refetchInterval: 5000,
    enabled: !!service_id,
  });

  const uptimeStats = useMemo(() => {
    const item = uptimeQuery.data;
    if (!item) return null;
    return {
      last24h: {
        percent: item.uptime_24h,
        activeHours: item.active_hours_24h,
        inactiveHours: item.inactive_hours_24h,
      },
      last7d: {
        percent: item.uptime_7d,
        activeHours: item.active_hours_7d,
        inactiveHours: item.inactive_hours_7d,
      },
      last30d: {
        percent: item.uptime_30d,
        activeHours: item.active_hours_30d,
        inactiveHours: item.inactive_hours_30d,
      },
    };
  }, [uptimeQuery.data]);

  if (uptimeQuery.isLoading) return <div className="p-4 text-xs text-base-content/60">Cargando métricas de uptime...</div>;
  if (uptimeQuery.isError) {
    return (
      <ErrorFallback
        error={uptimeQuery.error}
        onRetry={() => setTimeout(uptimeQuery.refetch)}
      />
    );
  }

  if (!uptimeStats) {
    return <div className="p-4 text-xs text-base-content/60">No se encontraron métricas de disponibilidad para esta instancia.</div>;
  }

  const getStatusColor = (percent: number) => {
    // 100% de uptime continuo es un riesgo (fallo de auto-sleep)
    if (percent === 100) return "text-warning border-warning/30 bg-warning/5";
    if (percent >= 99) return "text-success border-success/30 bg-success/5";
    if (percent >= 95) return "text-info border-info/30 bg-info/5";
    return "text-error border-error/30 bg-error/5";
  };

  const getRadialColorClass = (percent: number) => {
    if (percent === 100) return "text-warning";
    if (percent >= 99) return "text-success";
    if (percent >= 95) return "text-info";
    return "text-error";
  };

  const renderCard = (title: string, stats: { percent: number; activeHours: number; inactiveHours: number }) => {
    const cardBgColor = getStatusColor(stats.percent);
    const progressColor = getRadialColorClass(stats.percent);
    return (
      <div className={classNames("card border shadow-sm p-4 flex flex-col items-center justify-between text-center gap-4 transition-all hover:shadow-md", cardBgColor)}>
        <div className="flex flex-col items-center gap-0.5">
          <span className="text-[10px] uppercase font-bold tracking-wider text-base-content/60">{title}</span>
          {stats.percent === 100 && (
            <span className="badge badge-xs badge-warning border-warning/20 text-[8px] font-extrabold px-1 py-0.5 mt-0.5">
              ⚠️ RIESGO
            </span>
          )}
        </div>
        
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
            <span className="font-bold text-success">{stats.activeHours.toFixed(1)}h</span>
          </div>
          <div className="flex justify-between pt-0.5">
            <span>Tiempo Inactivo:</span>
            <span className="font-bold text-error">{stats.inactiveHours.toFixed(1)}h</span>
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
          El cálculo de uptime se lee directamente del motor analítico de base de datos SQL. 
          Un uptime continuo del <span className="font-bold text-warning">100.0%</span> representa un riesgo potencial en nuestro entorno de microservicios suspensibles, ya que indica que el auto-sleep del servicio no se ha activado.
        </p>
      </div>
    </div>
  );
};
