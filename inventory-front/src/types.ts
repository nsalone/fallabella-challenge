export type ProductStock = {
  sku: string;
  name: string;
  quantity: number;
};

export type Movement = {
  eventId: string;
  sku: string;
  type: "IN" | "OUT";
  quantity: number;
  occurredAt: string;
};
