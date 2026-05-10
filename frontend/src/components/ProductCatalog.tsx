'use client';

import { useState, useCallback, useEffect } from 'react';
import { Package, DollarSign, Boxes, AlertTriangle, RefreshCw } from 'lucide-react';

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

const API_URL = 'https://medstock-eewp.onrender.com';

export function ProductCatalog({ compact = false }: { compact?: boolean }) {
  const [products, setProducts] = useState<Product[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      const response = await fetch(`${API_URL}/api/v1/products`, {
        headers: { 'Authorization': 'Bearer test-api-key-123' }
      });

      if (response.ok) {
        const result = await response.json();
        setProducts(result.products || []);
        setError(null);

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
      setError('Failed to load. API offline.');
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 3000);
    return () => clearInterval(interval);
  }, [fetchData]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-2 border-red-600 border-t-transparent"></div>
        <span className="ml-3 text-zinc-500 font-medium">Loading products...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-zinc-900 text-red-400 p-4 rounded-xl border border-zinc-800 flex items-center gap-2 font-medium">
        <AlertTriangle className="w-5 h-5" />
        {error}
      </div>
    );
  }

  return (
    <div className="bg-zinc-900 rounded-2xl overflow-hidden border border-zinc-800">
      {/* Header with Stats */}
      {!compact && (
        <div className="bg-gradient-to-r from-red-900 to-red-950 p-6">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h2 className="text-2xl font-black flex items-center gap-2 text-white">
                <Package className="w-6 h-6" />
                Product Catalog
              </h2>
              <p className="text-red-300/60 text-sm mt-1 font-medium">Real-time inventory tracking</p>
            </div>
            <button
              onClick={fetchData}
              className="flex items-center gap-2 px-4 py-2 bg-white/10 hover:bg-white/20 rounded-lg transition-all text-sm font-bold text-white border border-white/20"
            >
              <RefreshCw className="w-4 h-4" />
              Refresh
            </button>
          </div>

          {stats && (
            <div className="grid grid-cols-4 gap-4">
              <div className="bg-black/30 backdrop-blur rounded-xl p-4 border border-red-800/50">
                <div className="flex items-center gap-2 text-red-300/60 text-xs font-bold uppercase tracking-wider mb-2">
                  <Package className="w-4 h-4" />
                  Total Products
                </div>
                <div className="text-3xl font-black text-white font-mono">{stats.total_products}</div>
              </div>
              <div className="bg-black/30 backdrop-blur rounded-xl p-4 border border-red-800/50">
                <div className="flex items-center gap-2 text-red-300/60 text-xs font-bold uppercase tracking-wider mb-2">
                  <Boxes className="w-4 h-4" />
                  Total Units
                </div>
                <div className="text-3xl font-black text-white font-mono">{stats.total_quantity.toLocaleString()}</div>
              </div>
              <div className="bg-black/30 backdrop-blur rounded-xl p-4 border border-red-800/50">
                <div className="flex items-center gap-2 text-red-300/60 text-xs font-bold uppercase tracking-wider mb-2">
                  <DollarSign className="w-4 h-4" />
                  Inventory Value
                </div>
                <div className="text-3xl font-black text-white font-mono">${stats.total_value.toLocaleString(undefined, { minimumFractionDigits: 2 })}</div>
              </div>
              <div className="bg-black/30 backdrop-blur rounded-xl p-4 border border-red-800/50">
                <div className="flex items-center gap-2 text-red-300/60 text-xs font-bold uppercase tracking-wider mb-2">
                  <AlertTriangle className="w-4 h-4" />
                  Low Stock
                </div>
                <div className="text-3xl font-black text-amber-400 font-mono">{stats.low_stock_count}</div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Compact Header */}
      {compact && (
        <div className="p-6 border-b border-zinc-800 bg-zinc-950">
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-black flex items-center gap-2 text-white">
              <Package className="w-5 h-5 text-red-500" />
              Recent Products
            </h2>
            <div className="flex items-center gap-4 text-xs">
              <span className="px-3 py-1 bg-red-600/20 text-red-400 rounded-full font-bold border border-red-600/30">
                {stats?.total_products || 0} items
              </span>
              <span className="text-zinc-500 font-mono">
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
            <div className="w-16 h-16 bg-zinc-800 rounded-full flex items-center justify-center mx-auto mb-4">
              <Boxes className="w-8 h-8 text-zinc-600" />
            </div>
            <h3 className="text-lg font-bold text-zinc-300 mb-2">No products yet</h3>
            <p className="text-zinc-600 font-medium">Upload inventory to see products here</p>
          </div>
        ) : (
          <div className="divide-y divide-zinc-800">
            {products.map((product, index) => {
              const itemValue = product.quantity * product.unit_price;
              const isLowStock = product.quantity < 50;

              return (
                <div
                  key={product.id || product.sku}
                  className="p-4 hover:bg-zinc-800/50 transition-all cursor-pointer group"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-4">
                      <div className="w-12 h-12 bg-gradient-to-br from-red-900/50 to-red-950/50 rounded-xl flex items-center justify-center border border-red-800/30">
                        <Package className="w-6 h-6 text-red-500" />
                      </div>
                      <div>
                        <h3 className="font-bold text-white group-hover:text-red-400 transition-colors">
                          {product.name}
                        </h3>
                        <div className="flex items-center gap-3 mt-1 text-xs">
                          <span className="bg-zinc-800 px-2 py-0.5 rounded font-mono text-zinc-400 border border-zinc-700">{product.sku}</span>
                          {product.category && (
                            <span className="text-zinc-500 font-medium">{product.category}</span>
                          )}
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center gap-6 text-right">
                      <div>
                        <div className={`flex items-center gap-1 font-bold font-mono ${isLowStock ? 'text-amber-400' : 'text-zinc-300'}`}>
                          <Boxes className="w-4 h-4" />
                          {product.quantity.toLocaleString()}
                        </div>
                        <div className="text-xs text-zinc-600 mt-0.5 uppercase tracking-wider">Qty</div>
                      </div>
                      <div>
                        <div className="flex items-center gap-1 text-red-400 font-bold font-mono">
                          <DollarSign className="w-4 h-4" />
                          {product.unit_price.toFixed(2)}
                        </div>
                        <div className="text-xs text-zinc-600 mt-0.5 uppercase tracking-wider">Price</div>
                      </div>
                      <div>
                        <div className="font-black text-white font-mono">
                          ${itemValue.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                        </div>
                        <div className="text-xs text-zinc-600 mt-0.5 uppercase tracking-wider">Value</div>
                      </div>
                    </div>
                  </div>

                  {isLowStock && (
                    <div className="mt-3 flex items-center gap-1 text-amber-400 text-xs bg-amber-400/10 px-2 py-1 rounded-lg w-fit border border-amber-400/20 font-bold">
                      <AlertTriangle className="w-3 h-3" />
                      Low stock alert ⚠️
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
        <div className="p-4 border-t border-zinc-800 bg-zinc-950 flex justify-between items-center">
          <span className="text-sm text-zinc-600 font-medium">
            Showing {products.length} of {stats?.total_products || 0} products
          </span>
          <div className="flex items-center gap-2">
            <button disabled className="px-4 py-2 border border-zinc-800 rounded-lg bg-zinc-900 text-zinc-600 font-bold text-sm cursor-not-allowed">
              ← Prev
            </button>
            <span className="px-4 py-2 text-zinc-500 font-mono text-sm">Page 1/1</span>
            <button disabled className="px-4 py-2 border border-zinc-800 rounded-lg bg-zinc-900 text-zinc-600 font-bold text-sm cursor-not-allowed">
              Next →
            </button>
          </div>
        </div>
      )}
    </div>
  );
}