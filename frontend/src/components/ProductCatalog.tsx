'use client';

import { useState, useCallback, useEffect } from 'react';
import { Package, DollarSign, Boxes, AlertTriangle, TrendingUp, RefreshCw } from 'lucide-react';

interface Product {
  id: string;
  sku: string;
  name: string;
  quantity: number;
  unit_price: number;
  category?: string;
}

interface Stats {
  total_products: number;
  total_quantity: number;
  total_value: number;
  low_stock_count: number;
}

export function ProductCatalog({ compact = false }: { compact?: boolean }) {
  const [products, setProducts] = useState<Product[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      const response = await fetch('http://localhost:8080/api/v1/products', {
        headers: { 'Authorization': 'Bearer test-api-key-123' }
      });

      if (response.ok) {
        const result = await response.json();
        setProducts(result.products || []);
        setError(null);

        // Calculate stats from products
        let totalValue = 0;
        let totalQuantity = 0;
        let lowStockCount = 0;

        result.products?.forEach((p: Product) => {
          totalValue += (p.unit_price || 0) * (p.quantity || 0);
          totalQuantity += p.quantity || 0;
          if ((p.quantity || 0) < 50) lowStockCount++;
        });

        setStats({
          total_products: result.products?.length || 0,
          total_quantity: totalQuantity,
          total_value: totalValue,
          low_stock_count: lowStockCount,
        });
        setIsLoading(false);
      } else {
        setError('Failed to fetch products');
        setIsLoading(false);
      }
    } catch (e) {
      setError('Failed to load products. Ensure API is running.');
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();

    // Refresh every 3 seconds
    const interval = setInterval(fetchData, 3000);

    return () => clearInterval(interval);
  }, [fetchData]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <span className="ml-3 text-gray-500">Loading products...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 text-red-600 p-4 rounded-lg flex items-center gap-2">
        <AlertTriangle className="w-5 h-5" />
        {error}
      </div>
    );
  }

  return (
    <div className="bg-white rounded-2xl shadow-lg overflow-hidden">
      {/* Header with Stats */}
      {!compact && (
        <div className="bg-gradient-to-r from-blue-600 to-indigo-600 p-6 text-white">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h2 className="text-2xl font-bold flex items-center gap-2">
                <Package className="w-6 h-6" />
                Product Catalog
              </h2>
              <p className="text-blue-100 mt-1">Real-time inventory tracking</p>
            </div>
            <button
              onClick={fetchData}
              className="flex items-center gap-2 px-4 py-2 bg-white/20 hover:bg-white/30 rounded-lg transition-all"
            >
              <RefreshCw className="w-4 h-4" />
              Refresh
            </button>
          </div>

          {stats && (
            <div className="grid grid-cols-4 gap-4">
              <div className="bg-white/10 rounded-xl p-4 backdrop-blur">
                <div className="flex items-center gap-2 text-blue-100 text-sm">
                  <Package className="w-4 h-4" />
                  Total Products
                </div>
                <div className="text-3xl font-bold mt-1">{stats.total_products}</div>
              </div>
              <div className="bg-white/10 rounded-xl p-4 backdrop-blur">
                <div className="flex items-center gap-2 text-blue-100 text-sm">
                  <Boxes className="w-4 h-4" />
                  Total Units
                </div>
                <div className="text-3xl font-bold mt-1">{stats.total_quantity.toLocaleString()}</div>
              </div>
              <div className="bg-white/10 rounded-xl p-4 backdrop-blur">
                <div className="flex items-center gap-2 text-blue-100 text-sm">
                  <DollarSign className="w-4 h-4" />
                  Inventory Value
                </div>
                <div className="text-3xl font-bold mt-1">${stats.total_value.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</div>
              </div>
              <div className="bg-white/10 rounded-xl p-4 backdrop-blur">
                <div className="flex items-center gap-2 text-blue-100 text-sm">
                  <AlertTriangle className="w-4 h-4" />
                  Low Stock
                </div>
                <div className="text-3xl font-bold mt-1">{stats.low_stock_count}</div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Compact Header */}
      {compact && (
        <div className="p-6 border-b bg-gradient-to-r from-gray-50 to-slate-50">
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-bold flex items-center gap-2 text-gray-800">
              <Package className="w-5 h-5 text-blue-600" />
              Recent Products
            </h2>
            <div className="flex items-center gap-4 text-sm">
              <span className="px-3 py-1 bg-blue-100 text-blue-700 rounded-full">
                {stats?.total_products || 0} items
              </span>
              <span className="text-gray-500">
                ${stats?.total_value?.toLocaleString(undefined, { minimumFractionDigits: 2 }) || '0.00'}
              </span>
            </div>
          </div>
        </div>
      )}

      {/* Product List */}
      <div className={compact ? "max-h-[400px] overflow-y-auto" : "max-h-[500px] overflow-y-auto"}>
        {products.length === 0 ? (
          <div className="p-12 text-center">
            <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <Boxes className="w-8 h-8 text-gray-400" />
            </div>
            <h3 className="text-lg font-medium text-gray-900 mb-2">No products yet</h3>
            <p className="text-gray-500">Upload inventory to see products here</p>
          </div>
        ) : (
          <div className="divide-y">
            {products.map((product, index) => {
              const itemValue = product.quantity * product.unit_price;
              const isLowStock = product.quantity < 50;

              return (
                <div
                  key={product.id || product.sku}
                  className="p-4 hover:bg-gradient-to-r hover:from-blue-50 hover:to-indigo-50 transition-all cursor-pointer group"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-4">
                      <div className="w-12 h-12 bg-gradient-to-br from-blue-100 to-indigo-100 rounded-xl flex items-center justify-center">
                        <Package className="w-6 h-6 text-blue-600" />
                      </div>
                      <div>
                        <h3 className="font-semibold text-gray-900 group-hover:text-blue-700 transition-colors">
                          {product.name}
                        </h3>
                        <div className="flex items-center gap-3 mt-1 text-sm">
                          <span className="bg-gray-100 px-2 py-0.5 rounded font-mono text-xs">{product.sku}</span>
                          {product.category && (
                            <span className="text-gray-500">{product.category}</span>
                          )}
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center gap-6 text-right">
                      <div>
                        <div className={`flex items-center gap-1 ${isLowStock ? 'text-amber-600' : 'text-gray-600'}`}>
                          <Boxes className="w-4 h-4" />
                          <span className="font-semibold">{product.quantity.toLocaleString()}</span>
                        </div>
                        <div className="text-xs text-gray-400 mt-0.5">Quantity</div>
                      </div>
                      <div>
                        <div className="flex items-center gap-1 text-green-600">
                          <DollarSign className="w-4 h-4" />
                          <span className="font-semibold">{product.unit_price.toFixed(2)}</span>
                        </div>
                        <div className="text-xs text-gray-400 mt-0.5">Unit Price</div>
                      </div>
                      <div>
                        <div className="font-bold text-gray-900">
                          ${itemValue.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                        </div>
                        <div className="text-xs text-gray-400 mt-0.5">Total Value</div>
                      </div>
                    </div>
                  </div>

                  {isLowStock && (
                    <div className="mt-3 flex items-center gap-1 text-amber-600 text-xs bg-amber-50 px-2 py-1 rounded-lg w-fit">
                      <AlertTriangle className="w-3 h-3" />
                      Low stock alert - Reorder soon
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Footer */}
      {!compact && products.length > 0 && (
        <div className="p-4 border-t bg-gray-50 flex justify-between items-center">
          <span className="text-sm text-gray-500">
            Showing {products.length} of {stats?.total_products || 0} products
          </span>
          <div className="flex items-center gap-2">
            <button disabled className="px-4 py-2 border rounded-lg bg-gray-100 text-gray-400 cursor-not-allowed">
              Previous
            </button>
            <span className="px-4 py-2">Page 1 of 1</span>
            <button disabled className="px-4 py-2 border rounded-lg bg-gray-100 text-gray-400 cursor-not-allowed">
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}