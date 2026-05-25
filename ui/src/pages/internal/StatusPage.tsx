import { useQuery } from "@tanstack/react-query";
import { statusService } from "../../services/status";
import { Cpu, HardDrive, Server, Activity, RefreshCcw } from "lucide-react";
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

  if (statusQuery.isLoading) {
    return (
      <div className="flex h-[50vh] w-full items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <span className="loading loading-ring loading-lg text-primary"></span>
          <span className="text-sm text-base-content/60 font-semibold animate-pulse">Obteniendo métricas del sistema...</span>
        </div>
      </div>
    );
  }

  if (statusQuery.isError) {
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
          <span>Error al cargar las métricas: {statusQuery.error?.message}</span>
        </div>
        <div className="flex-none">
          <button onClick={() => statusQuery.refetch()} className="btn btn-sm btn-ghost">
            Reintentar
          </button>
        </div>
      </div>
    );
  }

  const data = statusQuery.data;
  if (!data) return null;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Estado del Servidor</h2>
          <p className="text-sm text-base-content/60">Monitoreo de recursos de hardware en tiempo real.</p>
        </div>
        <button
          onClick={() => statusQuery.refetch()}
          className="btn btn-sm btn-ghost gap-2"
        >
          <RefreshCcw
            className={classNames("w-4 h-4", {
              "animate-spin": statusQuery.isFetching,
            })}
          />
          {statusQuery.isFetching ? "Actualizando..." : "Actualizado"}
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* CPU Usage Card */}
        <div className={classNames("card border bg-base-100 shadow-sm transition-all hover:shadow-md", getStatusColor(data.cpu.usage_percent))}>
          <div className="card-body items-center text-center p-6">
            <div className="flex items-center justify-between w-full mb-4">
              <span className="font-bold uppercase tracking-wider text-xs text-base-content/60 flex items-center gap-1.5">
                <Cpu className="w-4 h-4 text-primary" /> CPU
              </span>
              <span className="badge badge-sm font-semibold">{data.cpu.cores} Cores</span>
            </div>
            <div 
              className={classNames("radial-progress", getRadialColorClass(data.cpu.usage_percent))}
              style={{
                "--value": Math.round(data.cpu.usage_percent),
                "--size": "10rem",
                "--thickness": "12px",
              } as React.CSSProperties}
              role="progressbar"
            >
              <div className="flex flex-col items-center">
                <span className="text-3xl font-extrabold text-base-content">{Math.round(data.cpu.usage_percent)}%</span>
                <span className="text-[10px] text-base-content/50 uppercase font-semibold">En uso</span>
              </div>
            </div>
            <div className="mt-4 text-xs font-semibold text-base-content/70">
              Carga total del procesador
            </div>
          </div>
        </div>

        {/* RAM Memory Card */}
        <div className={classNames("card border bg-base-100 shadow-sm transition-all hover:shadow-md", getStatusColor(data.ram.usage_percent))}>
          <div className="card-body items-center text-center p-6">
            <div className="flex items-center justify-between w-full mb-4">
              <span className="font-bold uppercase tracking-wider text-xs text-base-content/60 flex items-center gap-1.5">
                <Activity className="w-4 h-4 text-secondary" /> RAM
              </span>
              <span className="badge badge-sm font-semibold">{formatBytes(data.ram.total_bytes, 0)} Total</span>
            </div>
            <div 
              className={classNames("radial-progress", getRadialColorClass(data.ram.usage_percent))}
              style={{
                "--value": Math.round(data.ram.usage_percent),
                "--size": "10rem",
                "--thickness": "12px",
              } as React.CSSProperties}
              role="progressbar"
            >
              <div className="flex flex-col items-center">
                <span className="text-3xl font-extrabold text-base-content">{Math.round(data.ram.usage_percent)}%</span>
                <span className="text-[10px] text-base-content/50 uppercase font-semibold">En uso</span>
              </div>
            </div>
            <div className="mt-4 text-xs text-base-content/75 flex flex-col gap-1 w-full font-medium">
              <div className="flex justify-between border-b border-base-200 py-1">
                <span>Usada:</span>
                <span className="font-bold">{formatBytes(data.ram.used_bytes)}</span>
              </div>
              <div className="flex justify-between py-1">
                <span>Disponible:</span>
                <span className="font-bold">{formatBytes(data.ram.free_bytes)}</span>
              </div>
            </div>
          </div>
        </div>

        {/* Storage Disk Card */}
        <div className={classNames("card border bg-base-100 shadow-sm transition-all hover:shadow-md", getStatusColor(data.disk.usage_percent))}>
          <div className="card-body items-center text-center p-6">
            <div className="flex items-center justify-between w-full mb-4">
              <span className="font-bold uppercase tracking-wider text-xs text-base-content/60 flex items-center gap-1.5">
                <HardDrive className="w-4 h-4 text-accent" /> Almacenamiento
              </span>
              <span className="badge badge-sm font-semibold">{formatBytes(data.disk.total_bytes, 0)} Total</span>
            </div>
            <div 
              className={classNames("radial-progress", getRadialColorClass(data.disk.usage_percent))}
              style={{
                "--value": Math.round(data.disk.usage_percent),
                "--size": "10rem",
                "--thickness": "12px",
              } as React.CSSProperties}
              role="progressbar"
            >
              <div className="flex flex-col items-center">
                <span className="text-3xl font-extrabold text-base-content">{Math.round(data.disk.usage_percent)}%</span>
                <span className="text-[10px] text-base-content/50 uppercase font-semibold">Lleno</span>
              </div>
            </div>
            <div className="mt-4 text-xs text-base-content/75 flex flex-col gap-1 w-full font-medium">
              <div className="flex justify-between border-b border-base-200 py-1">
                <span>Usado:</span>
                <span className="font-bold">{formatBytes(data.disk.used_bytes)}</span>
              </div>
              <div className="flex justify-between py-1">
                <span>Libre:</span>
                <span className="font-bold">{formatBytes(data.disk.free_bytes)}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* System Host Information */}
      <div className="card border bg-base-100 shadow-sm p-5">
        <div className="card-body p-0">
          <h3 className="font-bold text-sm uppercase tracking-wider text-base-content/65 mb-4 flex items-center gap-2">
            <Server className="w-4 h-4 text-primary" /> Información del Host
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-6 select-none">
            <div className="flex flex-col gap-1 p-3 bg-base-200/50 rounded-lg border border-base-200">
              <span className="text-[10px] uppercase font-bold text-base-content/50">Sistema Operativo</span>
              <span className="text-sm font-bold capitalize text-primary">{data.host.os}</span>
            </div>
            <div className="flex flex-col gap-1 p-3 bg-base-200/50 rounded-lg border border-base-200">
              <span className="text-[10px] uppercase font-bold text-base-content/50">Distribución / Plataforma</span>
              <span className="text-sm font-bold text-base-content truncate" title={data.host.platform}>
                {data.host.platform}
              </span>
            </div>
            <div className="flex flex-col gap-1 p-3 bg-base-200/50 rounded-lg border border-base-200">
              <span className="text-[10px] uppercase font-bold text-base-content/50">Uptime del Servidor</span>
              <span className="text-sm font-bold text-base-content">{formatUptime(data.host.uptime_seconds)}</span>
            </div>
            <div className="flex flex-col gap-1 p-3 bg-base-200/50 rounded-lg border border-base-200">
              <span className="text-[10px] uppercase font-bold text-base-content/50">Instancias Activas (PB)</span>
              <span className="text-sm font-bold text-secondary flex items-center gap-1.5">
                <span className="w-2.5 h-2.5 rounded-full bg-success animate-pulse inline-block"></span>
                {data.host.active_instances} corriendo
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
