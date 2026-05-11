import axios from 'axios';

const API_BASE_URL = 'http://localhost:8080/api/v1';

export interface MLStatus {
  enabled: boolean;
  last_update: string;
  model_version: string;
  techniques: string[];
}

export interface MLStatistics {
  total_requests: number;
  success_rate: number;
  average_latency: number;
  throughput_bps: number;
  error_rate: number;
  active_connections: number;
  timestamp: string;
}

export interface Technique {
  technique: string;
  effectiveness: number;
  sample_count: number;
  last_update: string;
  domain?: string;
}

export interface RequestLog {
  id: number;
  request_id: string;
  domain: string;
  method: string;
  status_code: number;
  success: boolean;
  latency: number;
  bytes_in: number;
  bytes_out: number;
  techniques: string;
  dpi_type: string;
  confidence: number;
  created_at: string;
  updated_at: string;
}

export interface FeedbackRequest {
  domain: string;
  technique: string;
  success: boolean;
}

export interface MLConfig {
  learning_rate: number;
  confidence_threshold: number;
  model_update_interval: string;
  effectiveness_threshold: number;
}

class ApiClient {
  private client = axios.create({
    baseURL: API_BASE_URL,
    timeout: 10000,
  });

  constructor() {
    this.client.interceptors.request.use((config) => {
      console.log(`API Request: ${config.method?.toUpperCase()} ${config.url}`);
      return config;
    });

    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        console.error('API Error:', error);
        throw error;
      }
    );
  }

  // ML Status and Statistics
  async getMLStatus(): Promise<MLStatus> {
    const response = await this.client.get('/ml/status');
    return response.data;
  }

  async getMLStatistics(): Promise<MLStatistics> {
    const response = await this.client.get('/ml/statistics');
    return response.data;
  }

  async getTechniques(domain?: string): Promise<Technique[]> {
    const params = domain ? { domain } : {};
    const response = await this.client.get('/ml/techniques', { params });
    return response.data;
  }

  async getHistory(hours: number): Promise<{ hours: number; statistics: MLStatistics[] }> {
    const response = await this.client.get(`/ml/history/${hours}`);
    return response.data;
  }

  // ML Control
  async postFeedback(feedback: FeedbackRequest): Promise<void> {
    await this.client.post('/ml/feedback', feedback);
  }

  async updateTechniqueEffectiveness(
    technique: string,
    effectiveness: number,
    domain?: string
  ): Promise<void> {
    await this.client.put(`/ml/techniques/${technique}/effectiveness`, {
      effectiveness,
    }, {
      params: domain ? { domain } : {},
    });
  }

  async retrainModel(force = false): Promise<void> {
    await this.client.post('/ml/retrain', { force });
  }

  async updateMLConfig(config: MLConfig): Promise<void> {
    await this.client.put('/ml/config', config);
  }

  // Request Tracing
  async getRequests(limit = 50): Promise<{ requests: RequestLog[]; count: number }> {
    const response = await this.client.get('/requests', {
      params: { limit },
    });
    return response.data;
  }

  async getRequestDetails(requestId: string): Promise<{ request: RequestLog; events: any[] }> {
    const response = await this.client.get(`/requests/${requestId}`);
    return response.data;
  }

  async getDomainStatistics(domain: string, hours = 24): Promise<any> {
    const response = await this.client.get(`/domains/${domain}/statistics`, {
      params: { hours },
    });
    return response.data;
  }

  // Health Check
  async healthCheck(): Promise<{ status: string; timestamp: string; version: string }> {
    const response = await this.client.get('/health');
    return response.data;
  }
}

export const apiClient = new ApiClient();
