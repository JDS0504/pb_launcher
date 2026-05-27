export interface OperationLog {
  id: string;
  created: string;
  operation: string;
  status: string;
  service: string;
  expand?: {
    service?: {
      id: string;
      name: string;
    };
  };
}

export interface UptimeStat {
  percent: number;
  activeMs: number;
  inactiveMs: number;
}

export interface UptimeStatsResult {
  last24h: UptimeStat;
  last7d: UptimeStat;
  last30d: UptimeStat;
}

export const calculateUptimeForLogs = (
  logs: OperationLog[],
  serviceId?: string,
  now = new Date(),
  serviceCreated?: string
): UptimeStatsResult => {
  const successfulLogs = logs
    .filter(l => l.status === "success" && (serviceId == null || l.service === serviceId))
    .sort((a, b) => new Date(a.created).getTime() - new Date(b.created).getTime());

  const calculateForDays = (days: number): UptimeStat => {
    let msInPeriod = days * 24 * 60 * 60 * 1000;
    let startTime = now.getTime() - msInPeriod;

    // Truncar al momento de creación si el servicio es más reciente que el inicio del periodo
    if (serviceCreated) {
      const createdTime = new Date(serviceCreated).getTime();
      if (createdTime > startTime) {
        startTime = createdTime;
        msInPeriod = now.getTime() - startTime;
      }
    }

    if (msInPeriod <= 0) {
      return { percent: 100, activeMs: 0, inactiveMs: 0 };
    }

    // 1. Determinar el estado inicial al inicio del periodo
    const logsBeforePeriod = successfulLogs.filter(
      l => new Date(l.created).getTime() < startTime
    );
    let isCurrentlyActive = false;
    if (logsBeforePeriod.length > 0) {
      const lastLogBefore = logsBeforePeriod[logsBeforePeriod.length - 1];
      isCurrentlyActive =
        lastLogBefore.operation === "start" || lastLogBefore.operation === "wakeup";
    } else {
      // Heurística si los logs están filtrados por API y no hay logs previos cargados
      const logsInPeriod = successfulLogs.filter(
        l => new Date(l.created).getTime() >= startTime
      );
      if (logsInPeriod.length > 0) {
        const firstLog = logsInPeriod[0];
        // Si el primer log registrado del periodo es apagar o suspender, venía encendido
        isCurrentlyActive = firstLog.operation === "stop" || firstLog.operation === "sleep";
      } else {
        isCurrentlyActive = false; // Por defecto asumimos inactivo si nunca hubo operaciones
      }
    }

    // 2. Filtrar logs que caen dentro del periodo
    const logsInPeriod = successfulLogs.filter(
      l => new Date(l.created).getTime() >= startTime
    );

    let totalActiveMs = 0;
    let lastEventTime = startTime;

    for (const log of logsInPeriod) {
      const logTime = new Date(log.created).getTime();
      if (isCurrentlyActive) {
        totalActiveMs += logTime - lastEventTime;
      }
      
      if (log.operation === "start" || log.operation === "wakeup") {
        isCurrentlyActive = true;
      } else if (log.operation === "stop" || log.operation === "sleep") {
        isCurrentlyActive = false;
      }
      lastEventTime = logTime;
    }

    if (isCurrentlyActive) {
      totalActiveMs += now.getTime() - lastEventTime;
    }

    const percent = Math.min(100, Math.max(0, (totalActiveMs / msInPeriod) * 100));
    return {
      percent,
      activeMs: totalActiveMs,
      inactiveMs: msInPeriod - totalActiveMs,
    };
  };

  return {
    last24h: calculateForDays(1),
    last7d: calculateForDays(7),
    last30d: calculateForDays(30),
  };
};
