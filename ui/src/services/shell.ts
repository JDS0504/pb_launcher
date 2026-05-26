import { pb } from "./client/pb";

/**
 * Construye la URL del WebSocket de la shell interactiva.
 * Convierte http → ws y https → wss automáticamente.
 * El token de auth se pasa como query param porque los WebSockets del
 * navegador no permiten cabeceras personalizadas durante el handshake.
 */
export const buildShellWsUrl = (): string => {
  const base = pb.baseURL.replace(/\/$/, "");
  const token = pb.authStore.token;

  let wsBase: string;
  if (base.startsWith("https://")) {
    wsBase = base.replace("https://", "wss://");
  } else if (base.startsWith("http://")) {
    wsBase = base.replace("http://", "ws://");
  } else {
    // Ruta relativa (modo embed) → usar ubicación actual del navegador
    const loc = window.location;
    const wsProtocol = loc.protocol === "https:" ? "wss:" : "ws:";
    wsBase = `${wsProtocol}//${loc.host}`;
  }

  return `${wsBase}/x-api/shell?token=${encodeURIComponent(token)}`;
};
