import { useQuery } from "@tanstack/react-query";
import { statusService } from "../../services/status";
import { serviceService } from "../../services/services";
import { useProxyConfigs } from "../../hooks/useProxyConfigs";
import { formatUrl } from "../../utils/url";
import { Cpu, HardDrive, Server, Activity, Power, ExternalLink } from "lucide-react";
import classNames from "classnames";
import React from "react";

const formatBytes = (bytes: number, decimals = 2) => {
  if (!bytes) return "0 B";
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ["B", "KB", "MB", "GB", "TB", "PB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
};

const formatUptime = (seconds: number) => {
  if (!seconds) return "0s";
  const d = Math.floor(seconds / (3600 * 24));
  const h = Math.floor((seconds % (3600 * 24)) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;

  const parts = [];
  if (d > 0) parts.push(`${d}d`);
  if (h > 0) parts.push(`${h}h`);
  if (m > 0) parts.push(`${m}m`);
  if (d === 0 && h === 0 && m === 0) parts.push(`${s}s`);
  return parts.join(" ");
};

export const StatusPage = () => {
  const statusQuery = useQuery({
    queryKey: ["system_status"],
    queryFn: statusService.fetchSystemStatus,
    refetchInterval: 3000,
  });

  const servicesQuery = useQuery({
    queryKey: ["services"],
    queryFn: serviceService.fetchAllServices,
    refetchInterval: 3000,
  });

  const proxy = useProxyConfigs();

  const getStatusColor = (percent: number) => {
    if (percent < 70) return "text-success border-success/30 bg-success/5";
    if (percent < 85) return "text-warning border-warning/30 bg-warning/5";
    return "text-error border-error/30 bg-error/5";
  };

  const getRadialColorClass = (percent: number) => {
    if (percent < 70) return "text-success";
    if (percent < 85) return "text-warning";
    return "text-error";
  };

  if (statusQuery.isLoading || servicesQuery.isLoading) {
    return (
      <div className="flex h-[50vh] w-full items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <span className="loading loading-ring loading-lg text-primary"></span>
          <span className="text-sm text-base-content/60 font-semibold animate-pulse">Obteniendo métricas del sistema...</span>
        </div>
      </div>
    );
  }

  if (statusQuery.isError || servicesQuery.isError) {
    return (
      <div className="alert alert-error shadow-lg">
        <div>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            className="stroke-current flex-shrink-0 h-6 w-6"
            fill="none"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth="2"
              d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <span>Error al cargar métricas de estado.</span>
        </div>
        <div className="flex-none">
          <button onClick={() => { statusQuery.refetch(); servicesQuery.refetch(); }} className="btn btn-sm btn-ghost">
            Reintentar
          </button>
        </div>
      </div>
    );
  }

  const data = statusQuery.data;
  const services = servicesQuery.data ?? [];
  const runningServices = services.filter(s => s.status === "running");

  if (!data) return null;

  return (
    <div className="space-y-6">
      {/* Cabecera (Sólo título sin iconos en móvil, oculto en PC) */}
      <h2 className="text-xl font-bold md:hidden block">Status</h2>

      <div className="grid grid-cols-3 gap-2 md:gap-6">
        {/* CPU Usage Card */}
        <div className={classNames("card border bg-base-100 shadow-sm transition-all hover:shadow-md p-3 md:p-4", getStatusColor(data.cpu.usage_percent))}>
          <div className="flex flex-col md:flex-row items-center gap-3 md:gap-4 w-full">
            <div 
              className={classNames("radial-progress shrink-0", getRadialColorClass(data.cpu.usage_percent))}
              style={{
                "--value": Math.round(data.cpu.usage_percent),
                "--size": "clamp(3.2rem, 10vw, 4.5rem)",
                "--thickness": "clamp(4px, 1vw, 7px)",
              } as React.CSSProperties}
              role="progressbar"
            >
              <span className="text-xs md:text-sm font-extrabold text-base-content">{Math.round(data.cpu.usage_percent)}%</span>
            </div>
            <div className="flex flex-col items-center md:items-start flex-1 min-w-0 w-full text-center md:text-left gap-1">
              <div className="flex items-center justify-between w-full gap-2">
                <span className="font-bold uppercase tracking-wider text-[10px] md:text-xs text-base-content/60 flex items-center gap-1">
                  <Cpu className="w-3.5 h-3.5 text-primary" /> CPU
                </span>
                <span className="badge badge-xs md:badge-sm font-semibold">{data.cpu.cores} Cores</span>
              </div>
              <p className="text-[10px] md:text-xs text-base-content/65 truncate hidden md:block">
                Carga total del procesador
              </p>
            </div>
          </div>
        </div>

        {/* RAM Memory Card */}
        <div className={classNames("card border bg-base-100 shadow-sm transition-all hover:shadow-md p-3 md:p-4", getStatusColor(data.ram.usage_percent))}>
          <div className="flex flex-col md:flex-row items-center gap-3 md:gap-4 w-full">
            <div 
              className={classNames("radial-progress shrink-0", getRadialColorClass(data.ram.usage_percent))}
              style={{
                "--value": Math.round(data.ram.usage_percent),
                "--size": "clamp(3.2rem, 10vw, 4.5rem)",
                "--thickness": "clamp(4px, 1vw, 7px)",
              } as React.CSSProperties}
              role="progressbar"
            >
              <span className="text-xs md:text-sm font-extrabold text-base-content">{Math.round(data.ram.usage_percent)}%</span>
            </div>
            <div className="flex flex-col items-center md:items-start flex-1 min-w-0 w-full text-center md:text-left gap-1">
              <div className="flex items-center justify-between w-full gap-2">
                <span className="font-bold uppercase tracking-wider text-[10px] md:text-xs text-base-content/60 flex items-center gap-1">
                  <Activity className="w-3.5 h-3.5 text-secondary" /> RAM
                </span>
                <span className="badge badge-xs md:badge-sm font-semibold">{formatBytes(data.ram.total_bytes, 0)}</span>
              </div>
              <div className="text-[9px] md:text-xs text-base-content/75 flex flex-col w-full font-medium">
                <div className="flex justify-between border-b border-base-200/50 py-0.5">
                  <span className="opacity-60">Usada:</span>
                  <span className="font-bold">{formatBytes(data.ram.used_bytes, 1)}</span>
                </div>
                <div className="flex justify-between py-0.5">
                  <span className="opacity-60">Libre:</span>
                  <span className="font-bold">{formatBytes(data.ram.free_bytes, 1)}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Storage Disk Card */}
        <div className={classNames("card border bg-base-100 shadow-sm transition-all hover:shadow-md p-3 md:p-4", getStatusColor(data.disk.usage_percent))}>
          <div className="flex flex-col md:flex-row items-center gap-3 md:gap-4 w-full">
            <div 
              className={classNames("radial-progress shrink-0", getRadialColorClass(data.disk.usage_percent))}
              style={{
                "--value": Math.round(data.disk.usage_percent),
                "--size": "clamp(3.2rem, 10vw, 4.5rem)",
                "--thickness": "clamp(4px, 1vw, 7px)",
              } as React.CSSProperties}
              role="progressbar"
            >
              <span className="text-xs md:text-sm font-extrabold text-base-content">{Math.round(data.disk.usage_percent)}%</span>
            </div>
            <div className="flex flex-col items-center md:items-start flex-1 min-w-0 w-full text-center md:text-left gap-1">
              <div className="flex items-center justify-between w-full gap-2">
                <span className="font-bold uppercase tracking-wider text-[10px] md:text-xs text-base-content/60 flex items-center gap-1">
                  <HardDrive className="w-3.5 h-3.5 text-accent" /> Disco
                </span>
                <span className="badge badge-xs md:badge-sm font-semibold">{formatBytes(data.disk.total_bytes, 0)}</span>
              </div>
              <div className="text-[9px] md:text-xs text-base-content/75 flex flex-col w-full font-medium">
                <div className="flex justify-between border-b border-base-200/50 py-0.5">
                  <span className="opacity-60">Usado:</span>
                  <span className="font-bold">{formatBytes(data.disk.used_bytes, 1)}</span>
                </div>
                <div className="flex justify-between py-0.5">
                  <span className="opacity-60">Libre:</span>
                  <span className="font-bold">{formatBytes(data.disk.free_bytes, 1)}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Instancias Corriendo con límites de recursos */}
      <div className="card border bg-base-100 shadow-sm p-4 md:p-5">
        <div className="card-body p-0 min-w-0">
          <div className="flex justify-between items-center mb-4">
            <h3 className="font-bold text-sm uppercase tracking-wider text-base-content/65 flex items-center gap-2">
              <Power className="w-4 h-4 text-success" /> Instancias Activas
            </h3>
            <span className="badge badge-sm badge-success font-semibold">{runningServices.length} Activas</span>
          </div>

          {runningServices.length === 0 ? (
            <div className="p-6 text-center text-xs text-base-content/50 bg-base-200/40 rounded-lg border border-dashed border-base-300">
              No hay instancias PocketBase activas en memoria actualmente.
            </div>
          ) : (
            <div className="overflow-x-auto min-w-0">
              <table className="table table-xs w-full text-xs font-mono">
                <thead>
                  <tr className="bg-base-200/50">
                    <th>Instancia</th>
                    <th className="text-right text-success">RAM</th>
                    <th className="text-right text-info">CPU</th>
                    <th className="hidden sm:table-cell">ID</th>
                    <th className="hidden sm:table-cell">Puerto</th>
                    <th className="text-right hidden sm:table-cell">Versión</th>
                  </tr>
                </thead>
                <tbody>
                  {runningServices.map(service => {
                    const stats = data.instances_stats?.[service.id];
                    return (
                      <tr key={service.id} className="hover:bg-base-200/35">
                        <td className="font-bold text-primary max-w-[120px] truncate" title={service.name}>
                          {proxy.base_domain ? (
                            <a
                              href={`${formatUrl(proxy.use_https ? "https" : "http", `${service.id}.${proxy.base_domain}`, proxy.use_https ? proxy.https_port : proxy.http_port)}/_/`}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="hover:underline flex items-center gap-1"
                            >
                              {service.name}
                              <ExternalLink className="w-3 h-3 opacity-50 shrink-0" />
                            </a>
                          ) : (
                            service.name
                          )}
                        </td>
                        <td className="text-right text-success font-semibold whitespace-nowrap">
                          {stats ? formatBytes(stats.memory_bytes) : "0 B"}
                        </td>
                        <td className="text-right text-info font-semibold whitespace-nowrap">
                          {stats ? `${stats.cpu_percent.toFixed(1)}%` : "0.0%"}
                        </td>
                        <td className="opacity-60 hidden sm:table-cell">{service.id}</td>
                        <td className="hidden sm:table-cell">
                          <span className="badge badge-sm badge-neutral font-mono">{service.port || "N/A"}</span>
                        </td>
                        <td className="text-right font-semibold hidden sm:table-cell">v{service.release_version}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>

      {/* System Host Information (Compactado) */}
      <div className="card border bg-base-100 shadow-sm p-4 md:p-5">
        <div className="card-body p-0">
          <h3 className="font-bold text-xs uppercase tracking-wider text-base-content/65 mb-3 flex items-center gap-2">
            <Server className="w-3.5 h-3.5 text-base-content/60" /> Servidor Host
          </h3>
          <div className="flex flex-wrap gap-4 text-xs font-mono text-base-content/75 bg-base-200/30 p-3 rounded-lg border border-base-200">
            <div>
              <span className="text-base-content/40 uppercase font-bold mr-1">OS:</span>
              <span className="font-bold text-primary">{data.host.os}</span>
            </div>
            <div className="hidden sm:inline-block opacity-25">|</div>
            <div>
              <span className="text-base-content/40 uppercase font-bold mr-1">Plataforma:</span>
              <span className="font-semibold">{data.host.platform}</span>
            </div>
            <div className="opacity-25">|</div>
            <div>
              <span className="text-base-content/40 uppercase font-bold mr-1">Uptime:</span>
              <span className="font-semibold text-secondary">{formatUptime(data.host.uptime_seconds)}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
