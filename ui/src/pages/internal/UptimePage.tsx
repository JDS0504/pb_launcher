import { useQuery } from "@tanstack/react-query";
import { useState, useMemo, type FC } from "react";
import classNames from "classnames";
import { Search, Activity, ExternalLink, ShieldCheck, AlertTriangle } from "lucide-react";
import { Link } from "react-router-dom";
import { serviceService, type ServiceDto } from "../../services/services";
import { calculateUptimeForLogs } from "../../utils/uptime";

const PAGE_SIZE = 10;

// Subcomponente de fila para renderizado de Uptime de forma diferida/optimizada por servicio
const ServiceUptimeRow: FC<{ service: ServiceDto; rangeDays: 1 | 7 | 30 }> = ({ service, rangeDays }) => {
  const operationsQuery = useQuery({
    queryKey: ["operation-logs", service.id],
    queryFn: () => serviceService.fetchOperationLogs(service.id),
    refetchInterval: 10000, // Menos frecuente para optimizar
  });

  const uptimeStat = useMemo(() => {
    if (!operationsQuery.data) return null;
    const stats = calculateUptimeForLogs(operationsQuery.data);
    if (rangeDays === 1) return stats.last24h;
    if (rangeDays === 7) return stats.last7d;
    return stats.last30d;
  }, [operationsQuery.data, rangeDays]);

  const percent = uptimeStat?.percent ?? 100; // Asumir 100% si no hay logs

  const getUptimeBadgeClass = (val: number) => {
    if (val >= 99) return "badge-success";
    if (val >= 95) return "badge-warning";
    return "badge-error";
  };

  return (
    <tr className="hover:bg-base-200/40">
      <td className="font-bold text-primary truncate max-w-[150px]">
        <Link to={`/services/${service.id}?section=uptime`} className="hover:underline flex items-center gap-1.5">
          {service.name} <ExternalLink className="w-3 h-3 opacity-50" />
        </Link>
      </td>
      <td>
        <span
          className={classNames("badge badge-xs", {
            "badge-success": service.status === "running",
            "badge-neutral": service.status === "stopped",
          })}
        >
          {service.status}
        </span>
      </td>
      <td>
        {operationsQuery.isLoading ? (
          <span className="loading loading-ring loading-xs text-base-content/40"></span>
        ) : (
          <span className={classNames("badge badge-sm font-mono font-bold", getUptimeBadgeClass(percent))}>
            {percent.toFixed(2)}%
          </span>
        )}
      </td>
      <td className="hidden sm:table-cell">
        <div className="w-full bg-base-300 rounded-full h-1.5 overflow-hidden max-w-[120px]">
          <div
            className={classNames("h-full rounded-full transition-all duration-500", {
              "bg-success": percent >= 99,
              "bg-warning": percent >= 95 && percent < 99,
              "bg-error": percent < 95,
            })}
            style={{ width: `${percent}%` }}
          />
        </div>
      </td>
      <td className="hidden md:table-cell text-xs font-mono text-base-content/65">
        {uptimeStat ? `${Math.round(uptimeStat.activeMs / 3600000)}h active` : "—"}
      </td>
    </tr>
  );
};

export const UptimePage: FC = () => {
  const [search, setSearch] = useState("");
  const [rangeDays, setRangeDays] = useState<1 | 7 | 30>(7);
  const [page, setPage] = useState(0);

  const servicesQuery = useQuery({
    queryKey: ["services"],
    queryFn: serviceService.fetchAllServices,
    refetchInterval: 5000,
  });

  const filtered = useMemo(() => {
    const servicesList = servicesQuery.data ?? [];
    return servicesList.filter(
      s =>
        s.name.toLowerCase().includes(search.toLowerCase()) ||
        s.id.toLowerCase().includes(search.toLowerCase())
    );
  }, [servicesQuery.data, search]);

  const totalPages = Math.ceil(filtered.length / PAGE_SIZE);
  const paginated = filtered.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  const handleSearch = (val: string) => {
    setSearch(val);
    setPage(0);
  };

  if (servicesQuery.isLoading) {
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
            <Activity className="w-5 h-5 text-primary animate-pulse" /> Uptime Monitor
          </h2>
          <p className="text-sm text-base-content/70">
            Control de disponibilidad global y diagnóstico de servicios activos.
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

      {/* Info General Consolidadas */}
      <div className="alert alert-info shadow-sm py-3 px-4 flex flex-row items-center gap-3">
        <ShieldCheck className="w-5 h-5 text-info shrink-0" />
        <div className="text-xs">
          Visualizando disponibilidad de <span className="font-bold">{filtered.length}</span> servicio(s) coincidente(s) en base al rango de <span className="font-bold">{rangeDays === 1 ? "24 horas" : `${rangeDays} días`}</span>.
        </div>
      </div>

      {/* Tabla Desglose */}
      <div className="card bg-base-100 border border-base-300 shadow-sm">
        <div className="card-body p-0 sm:p-4">
          {filtered.length === 0 ? (
            <div className="text-sm text-base-content/70 py-8 text-center flex flex-col items-center justify-center gap-2">
              <AlertTriangle className="w-6 h-6 text-warning" />
              <span>No se encontraron instancias de servicio creadas.</span>
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
                      <th className="hidden sm:table-cell">Disponibilidad</th>
                      <th className="hidden md:table-cell">Actividad</th>
                    </tr>
                  </thead>
                  <tbody>
                    {paginated.map(service => (
                      <ServiceUptimeRow
                        key={service.id}
                        service={service}
                        rangeDays={rangeDays}
                      />
                    ))}
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
