'use client';

import { useState, useCallback, useEffect } from 'react';
import { Upload, Plus, AlertCircle, CheckCircle2, Loader2, Trash2, X, RefreshCw } from 'lucide-react';

interface BulkItemInput {
  sku: string;
  name: string;
  description?: string;
  quantity: number;
  reserved_quantity?: number;
  unit_price: number;
  currency?: string;
  category?: string;
}

interface UploadResponse {
  session_id: string;
  status: string;
  total_items: number;
  processed: number;
  failed: number;
  progress: number;
  duration_ms: number;
  is_idempotent: boolean;
}

interface Product {
  id: string;
  sku: string;
  name: string;
  quantity: number;
  unit_price: number;
}

interface Stats {
  total_products: number;
  total_value: number;
  low_stock_count: number;
}

const API_URL = 'https://medstock-eewp.onrender.com';

export function BulkUploadPanel() {
  const [items, setItems] = useState<BulkItemInput[]>([]);
  const [currentItem, setCurrentItem] = useState<BulkItemInput>({
    sku: '',
    name: '',
    description: '',
    quantity: 0,
    unit_price: 0,
  });
  const [isUploading, setIsUploading] = useState(false);
  const [uploadResult, setUploadResult] = useState<UploadResponse | null>(null);
  const [recentProducts, setRecentProducts] = useState<Product[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);

  const fetchProducts = useCallback(async () => {
    try {
      const response = await fetch(`${API_URL}/api/v1/products`, {
        headers: { 'Authorization': 'Bearer test-api-key-123' }
      });
      if (response.ok) {
        const result = await response.json();
        setRecentProducts(result.products?.slice(0, 5) || []);

        let totalValue = 0;
        let lowStockCount = 0;
        result.products?.forEach((p: Product) => {
          totalValue += (p.unit_price || 0) * (p.quantity || 0);
          if ((p.quantity || 0) < 50) lowStockCount++;
        });

        setStats({
          total_products: result.products?.length || 0,
          total_value: totalValue,
          low_stock_count: lowStockCount,
        });
      }
    } catch (e) {
      console.error('Failed to fetch products:', e);
    }
  }, []);

  useEffect(() => {
    fetchProducts();
    const interval = setInterval(fetchProducts, 3000);
    return () => clearInterval(interval);
  }, [fetchProducts]);

  const addItem = useCallback(() => {
    if (currentItem.sku && currentItem.name && currentItem.quantity > 0 && currentItem.unit_price > 0) {
      setItems((prev) => [...prev, { ...currentItem }]);
      setCurrentItem({
        sku: '',
        name: '',
        description: '',
        quantity: 0,
        unit_price: 0,
      });
    }
  }, [currentItem]);

  const removeItem = useCallback((index: number) => {
    setItems((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const clearAll = useCallback(() => {
    setItems([]);
    setUploadResult(null);
  }, []);

  const handleUpload = useCallback(async () => {
    if (items.length === 0) return;

    setIsUploading(true);
    setUploadResult(null);

    const session_id = crypto.randomUUID();
    const idempotency_key = `${Date.now()}-${Math.random().toString(36).substring(7)}`;

    try {
      const response = await fetch(`${API_URL}/api/v1/bulk-upload`, {
        method: 'POST',
        headers: {
          'Authorization': 'Bearer test-api-key-123',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          session_id,
          idempotency_key,
          items,
        }),
      });

      if (response.ok) {
        const result = await response.json();
        setUploadResult(result);
        setTimeout(fetchProducts, 500);
      } else {
        const error = await response.json();
        alert(`Upload failed: ${error.error}`);
      }
    } catch (e) {
      alert('Upload failed: Network error');
    } finally {
      setIsUploading(false);
    }
  }, [items, fetchProducts]);

  const getStatusIcon = () => {
    if (!uploadResult) return null;
    switch (uploadResult.status) {
      case 'completed':
        return <CheckCircle2 className="w-6 h-6 text-red-500" />;
      case 'failed':
        return <AlertCircle className="w-6 h-6 text-zinc-400" />;
      case 'partial_failure':
        return <AlertCircle className="w-6 h-6 text-amber-500" />;
      default:
        return null;
    }
  };

  const canAddItem = currentItem.sku.length > 0 && currentItem.name.length > 0 && currentItem.quantity > 0 && currentItem.unit_price > 0;

  return (
    <div className="space-y-6">
      {/* Upload Form */}
      <div className="bg-zinc-900 rounded-2xl overflow-hidden border border-zinc-800">
        {/* Header */}
        <div className="bg-gradient-to-r from-red-700 to-red-900 p-6">
          <h2 className="text-xl font-black flex items-center gap-2 text-white">
            <Upload className="w-5 h-5" />
            Bulk Inventory Upload
          </h2>
          <p className="text-red-200 text-sm mt-1">Add items and upload to sync inventory</p>
        </div>

        <div className="p-6">
          <div className="grid grid-cols-2 gap-4 mb-6">
            <div>
              <label className="block text-xs font-bold text-zinc-400 uppercase tracking-wider mb-1">SKU *</label>
              <input
                type="text"
                value={currentItem.sku}
                onChange={(e) => setCurrentItem({ ...currentItem, sku: e.target.value.toUpperCase() })}
                className="w-full px-4 py-3 bg-zinc-800 border border-zinc-700 rounded-xl focus:ring-2 focus:ring-red-500 focus:border-transparent transition-all text-white placeholder-zinc-500 font-mono"
                placeholder="MED-001"
              />
            </div>
            <div>
              <label className="block text-xs font-bold text-zinc-400 uppercase tracking-wider mb-1">Product Name *</label>
              <input
                type="text"
                value={currentItem.name}
                onChange={(e) => setCurrentItem({ ...currentItem, name: e.target.value })}
                className="w-full px-4 py-3 bg-zinc-800 border border-zinc-700 rounded-xl focus:ring-2 focus:ring-red-500 focus:border-transparent transition-all text-white placeholder-zinc-500"
                placeholder="Medical Device Name"
              />
            </div>
            <div>
              <label className="block text-xs font-bold text-zinc-400 uppercase tracking-wider mb-1">Quantity *</label>
              <input
                type="number"
                value={currentItem.quantity || ''}
                onChange={(e) => setCurrentItem({ ...currentItem, quantity: parseInt(e.target.value) || 0 })}
                className="w-full px-4 py-3 bg-zinc-800 border border-zinc-700 rounded-xl focus:ring-2 focus:ring-red-500 focus:border-transparent transition-all text-white placeholder-zinc-500 font-mono"
                placeholder="0"
              />
            </div>
            <div>
              <label className="block text-xs font-bold text-zinc-400 uppercase tracking-wider mb-1">Unit Price ($) *</label>
              <input
                type="number"
                step="0.01"
                value={currentItem.unit_price || ''}
                onChange={(e) => setCurrentItem({ ...currentItem, unit_price: parseFloat(e.target.value) || 0 })}
                className="w-full px-4 py-3 bg-zinc-800 border border-zinc-700 rounded-xl focus:ring-2 focus:ring-red-500 focus:border-transparent transition-all text-white placeholder-zinc-500 font-mono"
                placeholder="0.00"
              />
            </div>
          </div>

          <button
            onClick={addItem}
            disabled={!canAddItem}
            className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-zinc-800 text-red-400 rounded-xl hover:bg-zinc-700 disabled:bg-zinc-900 disabled:text-zinc-600 transition-all font-bold text-sm border border-zinc-700"
          >
            <Plus className="w-5 h-5" />
            Add Item to Queue
          </button>
        </div>

        {/* Items Queue */}
        {items.length > 0 && (
          <div className="px-6 pb-4">
            <div className="flex items-center justify-between mb-3">
              <h3 className="font-bold text-zinc-300 text-sm uppercase tracking-wider">
                Items to Upload
                <span className="ml-2 px-2 py-0.5 bg-red-600/20 text-red-400 rounded text-xs font-mono">{items.length}</span>
              </h3>
              <button
                onClick={clearAll}
                className="text-xs text-zinc-500 hover:text-red-400 flex items-center gap-1 font-medium"
              >
                <Trash2 className="w-3 h-3" />
                Clear All
              </button>
            </div>
            <div className="max-h-48 overflow-y-auto border border-zinc-800 rounded-xl bg-zinc-950">
              {items.map((item, index) => (
                <div key={index} className="flex items-center justify-between p-3 border-b border-zinc-800 last:border-b-0 hover:bg-zinc-900">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 bg-red-600/20 rounded-lg flex items-center justify-center">
                      <span className="text-xs font-black text-red-400 font-mono">{index + 1}</span>
                    </div>
                    <div>
                      <span className="font-bold text-white font-mono text-sm">{item.sku}</span>
                      <span className="text-zinc-600 mx-2">→</span>
                      <span className="text-zinc-400">{item.name}</span>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="text-xs text-zinc-500 font-mono">Qty: {item.quantity}</span>
                    <span className="text-sm text-red-400 font-bold">${item.unit_price.toFixed(2)}</span>
                    <button
                      onClick={() => removeItem(index)}
                      className="text-zinc-600 hover:text-red-400 p-1 rounded hover:bg-zinc-800 transition-colors"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Upload Button */}
        <div className="px-6 pb-6">
          <button
            onClick={handleUpload}
            disabled={items.length === 0 || isUploading}
            className="w-full py-4 bg-gradient-to-r from-red-700 to-red-900 text-white rounded-xl hover:from-red-600 hover:to-red-800 disabled:from-zinc-700 disabled:to-zinc-800 disabled:cursor-not-allowed transition-all font-black text-lg shadow-lg shadow-red-900/50"
          >
            {isUploading ? (
              <span className="flex items-center justify-center gap-2">
                <Loader2 className="w-5 h-5 animate-spin" />
                Processing...
              </span>
            ) : (
              `Upload ${items.length} Item${items.length !== 1 ? 's' : ''} ⚡`
            )}
          </button>
        </div>

        {/* Upload Result */}
        {uploadResult && (
          <div className="px-6 pb-6">
            <div className="bg-zinc-950 rounded-xl p-4 border border-zinc-800">
              <div className="flex items-center gap-3 mb-4">
                {getStatusIcon()}
                <span className="font-black capitalize text-zinc-300 text-sm">
                  {uploadResult.status.replace(/_/g, ' ')}
                </span>
                {uploadResult.is_idempotent && (
                  <span className="text-xs bg-zinc-800 text-zinc-500 px-2 py-0.5 rounded font-mono">Idempotent</span>
                )}
              </div>
              <div className="grid grid-cols-3 gap-3 text-center">
                <div className="bg-zinc-900 rounded-lg p-3 border border-zinc-800">
                  <div className="text-2xl font-black text-white font-mono">{uploadResult.total_items}</div>
                  <div className="text-xs text-zinc-600 uppercase tracking-wider">Total</div>
                </div>
                <div className="bg-zinc-900 rounded-lg p-3 border border-zinc-800">
                  <div className="text-2xl font-black text-red-500 font-mono">{uploadResult.processed}</div>
                  <div className="text-xs text-zinc-600 uppercase tracking-wider">Success</div>
                </div>
                <div className="bg-zinc-900 rounded-lg p-3 border border-zinc-800">
                  <div className="text-2xl font-black text-zinc-400 font-mono">{uploadResult.failed}</div>
                  <div className="text-xs text-zinc-600 uppercase tracking-wider">Failed</div>
                </div>
              </div>
              <div className="mt-3 text-xs text-zinc-600 text-center font-mono">
                Processed in {uploadResult.duration_ms}ms ⚡
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Quick Stats */}
      {stats && stats.total_products > 0 && (
        <div className="bg-zinc-900 rounded-2xl border border-zinc-800 p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-black text-zinc-300 text-sm uppercase tracking-wider">Inventory Summary</h3>
            <button
              onClick={fetchProducts}
              className="text-xs text-red-400 hover:text-red-300 flex items-center gap-1 font-bold"
            >
              <RefreshCw className="w-3 h-3" />
              Refresh
            </button>
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div className="bg-zinc-950 rounded-xl p-4 text-center border border-zinc-800">
              <div className="text-3xl font-black text-red-500 font-mono">{stats.total_products}</div>
              <div className="text-xs text-zinc-600 uppercase tracking-wider mt-1">Products</div>
            </div>
            <div className="bg-zinc-950 rounded-xl p-4 text-center border border-zinc-800">
              <div className="text-3xl font-black text-white font-mono">
                ${stats.total_value.toLocaleString(undefined, { minimumFractionDigits: 2 })}
              </div>
              <div className="text-xs text-zinc-600 uppercase tracking-wider mt-1">Total Value</div>
            </div>
            <div className="bg-zinc-950 rounded-xl p-4 text-center border border-zinc-800">
              <div className="text-3xl font-black text-amber-500 font-mono">{stats.low_stock_count}</div>
              <div className="text-xs text-zinc-600 uppercase tracking-wider mt-1">Low Stock</div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}