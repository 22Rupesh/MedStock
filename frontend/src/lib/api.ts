import axios from 'axios';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
});

const DEFAULT_API_KEY = 'test-api-key-123';

api.interceptors.request.use((config) => {
  const apiKey = localStorage.getItem('api_key') || DEFAULT_API_KEY;
  config.headers.Authorization = `Bearer ${apiKey}`;
  return config;
});

export interface BulkItemInput {
  sku: string;
  name: string;
  description?: string;
  category?: string;
  manufacturer?: string;
  quantity: number;
  reserved_quantity: number;
  reorder_point?: number;
  warehouse_location?: string;
  unit_price: number;
  currency: string;
  bulk_price?: number;
}

export interface BulkUploadRequest {
  session_id: string;
  idempotency_key: string;
  items: BulkItemInput[];
}

export interface BulkUploadResponse {
  session_id: string;
  status: string;
  total_items: number;
  processed: number;
  failed: number;
  progress: number;
  duration_ms: number;
  is_idempotent: boolean;
}

export interface UploadStatus {
  id: string;
  session_id: string;
  client_id: string;
  idempotency_key: string;
  total_items: number;
  processed_items: number;
  failed_items: number;
  status: 'pending' | 'in_progress' | 'completed' | 'failed' | 'partial_failure';
  error_summary?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface Product {
  id: string;
  sku: string;
  client_id: string;
  name: string;
  description?: string;
  category?: string;
  manufacturer?: string;
  version: number;
  last_modified_hash?: string;
  created_at: string;
  updated_at: string;
  inventory?: {
    id: string;
    product_id: string;
    quantity: number;
    reserved_quantity: number;
    reorder_point?: number;
    warehouse_location?: string;
    status: string;
  };
  pricing?: {
    id: string;
    product_id: string;
    unit_price: number;
    currency: string;
    min_order_quantity: number;
    bulk_price?: number;
  };
}

export interface ProductListResponse {
  products: Product[];
  pagination: {
    page: number;
    page_size: number;
    total_items: number;
    total_pages: number;
  };
}

export const uploadApi = {
  bulkUpload: async (data: BulkUploadRequest): Promise<BulkUploadResponse> => {
    const response = await api.post<BulkUploadResponse>('/api/v1/bulk-upload', data);
    return response.data;
  },

  getStatus: async (sessionId: string): Promise<UploadStatus> => {
    const response = await api.get<UploadStatus>(`/api/v1/uploads/${sessionId}`);
    return response.data;
  },
};

export const catalogApi = {
  listProducts: async (page = 1, pageSize = 20): Promise<ProductListResponse> => {
    const response = await api.get<ProductListResponse>('/api/v1/products', {
      params: { page, page_size: pageSize },
    });
    return response.data;
  },

  getProduct: async (sku: string): Promise<Product> => {
    const response = await api.get<Product>(`/api/v1/products/${sku}`);
    return response.data;
  },
};