import type { Movement, ProductStock } from "./types";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

async function request<T>(path: string): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: "GET",
    headers: {
      Accept: "application/json",
    },
  });

  if (!response.ok) {
    const body = await response.json().catch(() => null);
    throw new Error(body?.error ?? `Request failed with status ${response.status}`);
  }

  return response.json() as Promise<T>;
}

export function getProductsStock(): Promise<ProductStock[]> {
  return request<ProductStock[]>("/products/stock");
}

export function getProductMovements(
  sku: string,
  limit = 100,
  offset = 0
): Promise<Movement[]> {
  const encodedSku = encodeURIComponent(sku);
  return request<Movement[]>(
    `/products/${encodedSku}/movements?limit=${limit}&offset=${offset}`
  );
}
