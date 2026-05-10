'use client';

import { QueryProvider } from '@/providers/QueryProvider';
import { BulkUploadPanel } from '@/components/BulkUploadPanel';
import { ProductCatalog } from '@/components/ProductCatalog';
import { Database, Boxes, Upload } from 'lucide-react';
import { useState } from 'react';

type TabType = 'upload' | 'catalog';

function AppShell() {
  const [activeTab, setActiveTab] = useState<TabType>('upload');

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-blue-50">
      <header className="bg-white/80 backdrop-blur-sm border-b border-gray-200 sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-gradient-to-br from-blue-600 to-indigo-600 rounded-xl flex items-center justify-center">
                <Boxes className="w-6 h-6 text-white" />
              </div>
              <div>
                <h1 className="text-2xl font-bold text-gray-900">MedStock</h1>
                <p className="text-xs text-gray-500">Inventory Sync Engine</p>
              </div>
            </div>
            <div className="flex items-center gap-2 px-3 py-1 bg-green-100 text-green-700 rounded-full text-sm">
              <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
              System Online
            </div>
          </div>
        </div>
      </header>

      <nav className="bg-white border-b">
        <div className="max-w-7xl mx-auto px-4">
          <div className="flex gap-1">
            <button
              onClick={() => setActiveTab('upload')}
              className={`flex items-center gap-2 px-4 py-3 font-medium transition-all border-b-2 ${
                activeTab === 'upload'
                  ? 'text-blue-600 border-blue-600'
                  : 'text-gray-500 border-transparent hover:text-gray-700'
              }`}
            >
              <Upload className="w-4 h-4" />
              Upload
            </button>
            <button
              onClick={() => setActiveTab('catalog')}
              className={`flex items-center gap-2 px-4 py-3 font-medium transition-all border-b-2 ${
                activeTab === 'catalog'
                  ? 'text-blue-600 border-blue-600'
                  : 'text-gray-500 border-transparent hover:text-gray-700'
              }`}
            >
              <Database className="w-4 h-4" />
              Catalog
            </button>
          </div>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-4 py-8">
        {activeTab === 'upload' ? (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            <BulkUploadPanel />
            <ProductCatalog compact />
          </div>
        ) : (
          <ProductCatalog />
        )}
      </main>
    </div>
  );
}

export default function Home() {
  return (
    <QueryProvider>
      <AppShell />
    </QueryProvider>
  );
}