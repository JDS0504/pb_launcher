import { useQuery } from "@tanstack/react-query";
import { useState, useMemo, type FC } from "react";
import classNames from "classnames";
import { Search, Activity, ExternalLink, AlertTriangle } from "lucide-react";
import { Link } from "react-router-dom";
import { serviceService, type ServiceUptimeViewDto } from "../../services/services";

const PAGE_SIZE = 10;

// Renderizar indicador visual de Uptime e información de advertencia de riesgo
const UptimePercentageBadge: FC<{ percent: number }> = ({ percent }) => {
  const getUptimeBadgeClass = (val: number) => {
    // Si es 100% de uptime, es un riesgo para nuestro caso porque no está haciendo el auto-sleep
    if (val === 100) return "badge-warning border-warning/30 bg-warning/5 text-warning";
    if (val >= 99) return "badge-success border-success/30 bg-success/5 text-success";
    if (val >= 95) return "badge-info border-info/30 bg-info/5 text-info";
    return "badge-error border-error/30 bg-error/5 text-error";
  };

  return (
    <div className="flex items-center gap-1.5">
      <span className={classNames("badge badge-sm font-mono font-bold px-2.5 py-1.5 border", getUptimeBadgeClass(percent))}>
        {percent.toFixed(2)}%
      </span>
      {percent === 100 && (
        <span className="text-[9px] font-bold text-warning uppercase tracking-wider scale-90 origin-left" title="Riesgo: 100% Uptime detectado (posible fallo de auto-sleep)">
          ⚠️ Riesgo
        </span>
      )}
    </div>
  );
};

export const UptimePage: FC = () => {
  const [search, setSearch] = useState("");
  const [rangeDays, setRangeDays] = useState<1 | 7 | 30>(7);
  const [page, setPage] = useState(0);

  // Cargar datos consolidados desde la vista SQL
  const uptimeQuery = useQuery({
    queryKey: ["service-uptime-view"],
    queryFn: serviceService.fetchServiceUptimeView,
    refetchInterval: 5000,
  });

  const handleSearch = (val: string) => {
    setSearch(val);
    setPage(0);
  };

  // Filtrar, ordenar y paginar la data en el frontend
  const processedData = useMemo(() => {
    const list = uptimeQuery.data ?? [];
    
    // 1. Filtrar según la búsqueda
    const filtered = list.filter(
      item =>
        item.service_name.toLowerCase().includes(search.toLowerCase()) ||
        item.id.toLowerCase().includes(search.toLowerCase())
    );

    // 2. Ordenar de mayor a menor según las horas activas de la última semana (active_hours_7d)
    return filtered.sort((a, b) => b.active_hours_7d - a.active_hours_7d);
  }, [uptimeQuery.data, search]);

  const totalPages = Math.ceil(processedData.length / PAGE_SIZE);
  const paginated = processedData.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  if (uptimeQuery.isLoading) {
    return (
      <div className="flex h-[50vh] w-full items-center justify-center">
        <span className="loading loading-ring loading-lg text-primary"></span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Cabecera */}
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-3.5">
        <div>
          <h2 className="text-xl font-semibold flex items-center gap-2">
            <Activity className="w-5 h-5 text-primary animate-pulse" /> Monitor de Disponibilidad Global
          </h2>
          <p className="text-sm text-base-content/70">
            Línea de tiempo consolidada y diagnóstico automático de consumo.
          </p>
        </div>

        {/* Filtros */}
        <div className="flex flex-col sm:flex-row gap-2 w-full lg:w-auto items-stretch sm:items-center">
          <div className="relative flex-grow sm:flex-none">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-base-content/40" />
            <input
              id="uptime-global-search"
              type="search"
              className="input input-bordered input-sm w-full sm:w-56 pl-9"
              placeholder="Buscar servicio..."
              value={search}
              onChange={e => handleSearch(e.target.value)}
            />
          </div>

          <div className="flex gap-1 shrink-0 justify-center sm:justify-start">
            {([1, 7, 30] as const).map(d => (
              <button
                key={d}
                className={classNames("btn btn-xs capitalize", {
                  "btn-neutral": rangeDays === d,
                  "btn-ghost": rangeDays !== d,
                })}
                onClick={() => setRangeDays(d)}
              >
                {d === 1 ? "24 Horas" : `${d} Días`}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Explicación de Riesgo de 100% Uptime */}
      <div className="alert alert-warning border border-warning/20 bg-warning/5 text-warning-content/90 text-xs py-2 px-3 rounded-lg shadow-sm flex items-start gap-2">
        <AlertTriangle className="w-4 h-4 shrink-0 text-warning" />
        <div>
          <span className="font-bold">Nota de Operación:</span> Los servicios están diseñados para suspenderse (auto-sleep) por inactividad. Un uptime continuo de 100% representa un riesgo potencial de consumo continuo de recursos.
        </div>
      </div>

      {/* Tabla Desglose */}
      <div className="card bg-base-100 border border-base-300 shadow-sm">
        <div className="card-body p-0 sm:p-4">
          {processedData.length === 0 ? (
            <div className="text-sm text-base-content/70 py-8 text-center flex flex-col items-center justify-center gap-2">
              <AlertTriangle className="w-6 h-6 text-warning" />
              <span>No se encontraron instancias de servicio registradas.</span>
            </div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="table table-sm w-full">
                  <thead>
                    <tr className="bg-base-200/50">
                      <th>Instancia</th>
                      <th>Estado</th>
                      <th>Uptime ({rangeDays === 1 ? "24h" : `${rangeDays}d`})</th>
                      <th>Horas Activas</th>
                      <th>Horas Inactivas</th>
                    </tr>
                  </thead>
                  <tbody>
                    {paginated.map((item: ServiceUptimeViewDto) => {
                      const percent = rangeDays === 1 ? item.uptime_24h : rangeDays === 7 ? item.uptime_7d : item.uptime_30d;
                      const activeH = rangeDays === 1 ? item.active_hours_24h : rangeDays === 7 ? item.active_hours_7d : item.active_hours_30d;
                      const inactiveH = rangeDays === 1 ? item.inactive_hours_24h : rangeDays === 7 ? item.inactive_hours_7d : item.inactive_hours_30d;

                      return (
                        <tr key={item.id} className="hover:bg-base-200/40">
                          <td className="font-bold text-primary truncate max-w-[150px]">
                            <Link to={`/services/${item.id}?section=uptime`} className="hover:underline flex items-center gap-1.5">
                              {item.service_name} <ExternalLink className="w-3 h-3 opacity-50" />
                            </Link>
                          </td>
                          <td>
                            <span
                              className={classNames("badge badge-xs", {
                                "badge-success": item.service_status === "running",
                                "badge-neutral": item.service_status === "stopped",
                                "badge-primary": item.service_status === "sleeping",
                              })}
                            >
                              {item.service_status}
                            </span>
                          </td>
                          <td>
                            <UptimePercentageBadge percent={percent} />
                          </td>
                          <td className="font-mono text-xs text-success font-semibold">
                            {activeH.toFixed(1)}h
                          </td>
                          <td className="font-mono text-xs text-error font-semibold">
                            {inactiveH.toFixed(1)}h
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>

              {/* Paginación */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between gap-2 p-3 sm:p-0 sm:pt-4 border-t border-base-200 mt-2">
                  <span className="text-xs text-base-content/50">
                    Página {page + 1} de {totalPages}
                  </span>
                  <div className="flex gap-1">
                    <button
                      className="btn btn-xs btn-ghost"
                      disabled={page === 0}
                      onClick={() => setPage(p => p - 1)}
                    >
                      ← Anterior
                    </button>
                    <button
                      className="btn btn-xs btn-ghost"
                      disabled={page >= totalPages - 1}
                      onClick={() => setPage(p => p + 1)}
                    >
                      Siguiente →
                    </button>
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
};
