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
    <div className="min-h-screen bg-black text-white">
      {/* Header */}
      <header className="bg-zinc-950 border-b border-zinc-800 sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-11 h-11 bg-gradient-to-br from-red-600 to-red-800 rounded-xl flex items-center justify-center shadow-lg shadow-red-900/50">
                <Boxes className="w-6 h-6 text-white" />
              </div>
              <div>
                <h1 className="text-2xl font-black tracking-tight">MedStock</h1>
                <p className="text-xs text-zinc-500">Inventory Sync Engine</p>
              </div>
            </div>
            <div className="flex items-center gap-2 px-3 py-1.5 bg-red-600/20 border border-red-600/50 rounded-full text-xs font-medium text-red-400">
              <span className="w-2 h-2 bg-red-500 rounded-full animate-pulse"></span>
              System Online
            </div>
          </div>
        </div>
      </header>

      {/* Navigation */}
      <nav className="bg-zinc-950 border-b border-zinc-800">
        <div className="max-w-7xl mx-auto px-4">
          <div className="flex gap-1">
            <button
              onClick={() => setActiveTab('upload')}
              className={`flex items-center gap-2 px-5 py-4 font-bold text-sm transition-all border-b-2 ${
                activeTab === 'upload'
                  ? 'text-red-500 border-red-500'
                  : 'text-zinc-500 border-transparent hover:text-white'
              }`}
            >
              <Upload className="w-4 h-4" />
              Upload
            </button>
            <button
              onClick={() => setActiveTab('catalog')}
              className={`flex items-center gap-2 px-5 py-4 font-bold text-sm transition-all border-b-2 ${
                activeTab === 'catalog'
                  ? 'text-red-500 border-red-500'
                  : 'text-zinc-500 border-transparent hover:text-white'
              }`}
            >
              <Database className="w-4 h-4" />
              Catalog
            </button>
          </div>
        </div>
      </nav>

      {/* Main Content */}
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

      {/* Footer */}
      <footer className="border-t border-zinc-900 py-6 mt-12">
        <div className="max-w-7xl mx-auto px-4 text-center text-zinc-600 text-xs">
          <p>MedStock © 2024 • Built with Go, Next.js & PostgreSQL</p>
        </div>
      </footer>
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