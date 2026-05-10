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
      const response = await fetch('http://localhost:8080/api/v1/products', {
        headers: { 'Authorization': 'Bearer test-api-key-123' }
      });
      if (response.ok) {
        const result = await response.json();
        setRecentProducts(result.products?.slice(0, 5) || []);

        // Calculate stats
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
      const response = await fetch('http://localhost:8080/api/v1/bulk-upload', {
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
        // Refresh products after upload
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
        return <CheckCircle2 className="w-6 h-6 text-green-500" />;
      case 'failed':
        return <AlertCircle className="w-6 h-6 text-red-500" />;
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
      <div className="bg-white rounded-2xl shadow-lg overflow-hidden">
        <div className="bg-gradient-to-r from-indigo-600 to-purple-600 p-6 text-white">
          <h2 className="text-xl font-bold flex items-center gap-2">
            <Upload className="w-5 h-5" />
            Bulk Inventory Upload
          </h2>
          <p className="text-indigo-100 text-sm mt-1">Add items and upload to sync inventory</p>
        </div>

        <div className="p-6">
          <div className="grid grid-cols-2 gap-4 mb-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">SKU *</label>
              <input
                type="text"
                value={currentItem.sku}
                onChange={(e) => setCurrentItem({ ...currentItem, sku: e.target.value.toUpperCase() })}
                className="w-full px-4 py-2.5 border border-gray-200 rounded-xl focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-all"
                placeholder="MED-001"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Product Name *</label>
              <input
                type="text"
                value={currentItem.name}
                onChange={(e) => setCurrentItem({ ...currentItem, name: e.target.value })}
                className="w-full px-4 py-2.5 border border-gray-200 rounded-xl focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-all"
                placeholder="Medical Device Name"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Quantity *</label>
              <input
                type="number"
                value={currentItem.quantity || ''}
                onChange={(e) => setCurrentItem({ ...currentItem, quantity: parseInt(e.target.value) || 0 })}
                className="w-full px-4 py-2.5 border border-gray-200 rounded-xl focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-all"
                placeholder="0"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Unit Price ($) *</label>
              <input
                type="number"
                step="0.01"
                value={currentItem.unit_price || ''}
                onChange={(e) => setCurrentItem({ ...currentItem, unit_price: parseFloat(e.target.value) || 0 })}
                className="w-full px-4 py-2.5 border border-gray-200 rounded-xl focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-all"
                placeholder="0.00"
              />
            </div>
          </div>

          <button
            onClick={addItem}
            disabled={!canAddItem}
            className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-blue-50 text-blue-600 rounded-xl hover:bg-blue-100 disabled:bg-gray-100 disabled:text-gray-400 transition-all font-medium"
          >
            <Plus className="w-5 h-5" />
            Add Item to Queue
          </button>
        </div>

        {/* Items Queue */}
        {items.length > 0 && (
          <div className="px-6 pb-4">
            <div className="flex items-center justify-between mb-3">
              <h3 className="font-semibold text-gray-800">
                Items to Upload
                <span className="ml-2 px-2 py-0.5 bg-indigo-100 text-indigo-700 rounded-full text-sm">{items.length}</span>
              </h3>
              <button
                onClick={clearAll}
                className="text-sm text-red-500 hover:text-red-700 flex items-center gap-1"
              >
                <Trash2 className="w-4 h-4" />
                Clear All
              </button>
            </div>
            <div className="max-h-48 overflow-y-auto border border-gray-200 rounded-xl">
              {items.map((item, index) => (
                <div key={index} className="flex items-center justify-between p-3 border-b last:border-b-0 hover:bg-gray-50">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 bg-indigo-100 rounded-lg flex items-center justify-center">
                      <span className="text-xs font-bold text-indigo-600">{index + 1}</span>
                    </div>
                    <div>
                      <span className="font-medium text-gray-900">{item.sku}</span>
                      <span className="text-gray-400 mx-2">-</span>
                      <span className="text-gray-600">{item.name}</span>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="text-sm text-gray-500">Qty: {item.quantity}</span>
                    <span className="text-sm text-green-600">${item.unit_price.toFixed(2)}</span>
                    <button
                      onClick={() => removeItem(index)}
                      className="text-gray-400 hover:text-red-500 p-1 rounded hover:bg-red-50 transition-colors"
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
            className="w-full py-4 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-xl hover:from-indigo-700 hover:to-purple-700 disabled:from-gray-400 disabled:to-gray-400 disabled:cursor-not-allowed transition-all font-semibold text-lg shadow-lg shadow-indigo-500/25"
          >
            {isUploading ? (
              <span className="flex items-center justify-center gap-2">
                <Loader2 className="w-5 h-5 animate-spin" />
                Processing...
              </span>
            ) : (
              `Upload ${items.length} Item${items.length !== 1 ? 's' : ''}`
            )}
          </button>
        </div>

        {/* Upload Result */}
        {uploadResult && (
          <div className="px-6 pb-6">
            <div className="bg-gradient-to-r from-gray-50 to-slate-50 rounded-xl p-4 border border-gray-200">
              <div className="flex items-center gap-3 mb-4">
                {getStatusIcon()}
                <span className="font-semibold capitalize text-gray-800">
                  {uploadResult.status.replace(/_/g, ' ')}
                </span>
                {uploadResult.is_idempotent && (
                  <span className="text-xs bg-amber-100 text-amber-700 px-2 py-0.5 rounded">Idempotent</span>
                )}
              </div>
              <div className="grid grid-cols-3 gap-4 text-center">
                <div className="bg-white rounded-lg p-3 shadow-sm">
                  <div className="text-2xl font-bold text-gray-900">{uploadResult.total_items}</div>
                  <div className="text-xs text-gray-500">Total</div>
                </div>
                <div className="bg-white rounded-lg p-3 shadow-sm">
                  <div className="text-2xl font-bold text-green-600">{uploadResult.processed}</div>
                  <div className="text-xs text-gray-500">Success</div>
                </div>
                <div className="bg-white rounded-lg p-3 shadow-sm">
                  <div className="text-2xl font-bold text-red-600">{uploadResult.failed}</div>
                  <div className="text-xs text-gray-500">Failed</div>
                </div>
              </div>
              <div className="mt-3 text-xs text-gray-500 text-center">
                Processed in {uploadResult.duration_ms}ms
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Quick Stats */}
      {stats && stats.total_products > 0 && (
        <div className="bg-white rounded-2xl shadow-lg p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold text-gray-800">Inventory Summary</h3>
            <button
              onClick={fetchProducts}
              className="text-sm text-indigo-600 hover:text-indigo-800 flex items-center gap-1"
            >
              <RefreshCw className="w-4 h-4" />
              Refresh
            </button>
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div className="bg-gradient-to-br from-blue-50 to-indigo-50 rounded-xl p-4 text-center">
              <div className="text-3xl font-bold text-blue-600">{stats.total_products}</div>
              <div className="text-sm text-gray-500">Products</div>
            </div>
            <div className="bg-gradient-to-br from-green-50 to-emerald-50 rounded-xl p-4 text-center">
              <div className="text-3xl font-bold text-green-600">
                ${stats.total_value.toLocaleString(undefined, { minimumFractionDigits: 2 })}
              </div>
              <div className="text-sm text-gray-500">Total Value</div>
            </div>
            <div className="bg-gradient-to-br from-amber-50 to-orange-50 rounded-xl p-4 text-center">
              <div className="text-3xl font-bold text-amber-600">{stats.low_stock_count}</div>
              <div className="text-sm text-gray-500">Low Stock</div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}