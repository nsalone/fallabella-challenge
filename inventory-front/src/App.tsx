import { useEffect, useState } from "react";
import { getProductMovements, getProductsStock } from "./api";
import type { Movement, ProductStock } from "./types";

function App() {
  const [products, setProducts] = useState<ProductStock[]>([]);
  const [selectedProduct, setSelectedProduct] = useState<ProductStock | null>(null);
  const [movements, setMovements] = useState<Movement[]>([]);

  const [loadingProducts, setLoadingProducts] = useState(false);
  const [loadingMovements, setLoadingMovements] = useState(false);
  const [error, setError] = useState("");

  async function loadProducts() {
    setLoadingProducts(true);
    setError("");

    try {
      const data = await getProductsStock();
      setProducts(data);

      if (!selectedProduct && data.length > 0) {
        await selectProduct(data[0]);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unexpected error loading products");
    } finally {
      setLoadingProducts(false);
    }
  }

  async function selectProduct(product: ProductStock) {
    setSelectedProduct(product);
    setLoadingMovements(true);
    setError("");

    try {
      const data = await getProductMovements(product.sku, 100, 0);
      setMovements(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unexpected error loading movements");
    } finally {
      setLoadingMovements(false);
    }
  }

  useEffect(() => {
    loadProducts();
  }, []);

  return (
    <main className="container">
      <header className="header">
        <div>
          <h1>Inventory Dashboard</h1>
          <p>Current stock and movement history.</p>
        </div>

        <button onClick={loadProducts}>Refresh</button>
      </header>

      {error && <p className="error">{error}</p>}

      <section className="card">
        <h2>Products stock</h2>

        {loadingProducts ? (
          <p className="info">Loading products...</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>SKU</th>
                <th>Name</th>
                <th>Current stock</th>
              </tr>
            </thead>
            <tbody>
              {products.map((product) => (
                <tr
                  key={product.sku}
                  onClick={() => selectProduct(product)}
                  className={selectedProduct?.sku === product.sku ? "selected" : ""}
                >
                  <td>{product.sku}</td>
                  <td>{product.name}</td>
                  <td>{product.quantity}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      {selectedProduct && (
        <section className="card">
          <h2>Movement history - {selectedProduct.sku}</h2>

          {loadingMovements ? (
            <p className="info">Loading movements...</p>
          ) : movements.length === 0 ? (
            <p>No movements found.</p>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Event ID</th>
                  <th>Type</th>
                  <th>Quantity</th>
                  <th>Occurred at</th>
                </tr>
              </thead>
              <tbody>
                {movements.map((movement) => (
                  <tr key={movement.eventId}>
                    <td>{movement.eventId}</td>
                    <td>
                      <span className={movement.type === "IN" ? "badge in" : "badge out"}>
                        {movement.type}
                      </span>
                    </td>
                    <td>{movement.quantity}</td>
                    <td>{formatDate(movement.occurredAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      )}
    </main>
  );
}

function formatDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString();
}

export default App;
