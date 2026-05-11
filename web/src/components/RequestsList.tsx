import React, { useState, useEffect } from 'react';
import { apiClient, type RequestLog } from '../api/client';
import { Eye, CheckCircle, XCircle, Clock } from 'lucide-react';

interface RequestsListProps {
  limit?: number;
  compact?: boolean;
  className?: string;
}

export const RequestsList: React.FC<RequestsListProps> = ({ 
  limit = 50, 
  compact = false, 
  className = '' 
}) => {
  const [requests, setRequests] = useState<RequestLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchRequests = async () => {
      try {
        setLoading(true);
        const data = await apiClient.getRequests(limit);
        setRequests(data.requests);
        setError(null);
      } catch (err) {
        setError('Failed to fetch requests');
        console.error('Error fetching requests:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchRequests();
  }, [limit]);

  const formatLatency = (ms: number): string => {
    if (ms >= 1000) {
      return (ms / 1000).toFixed(1) + 's';
    }
    return ms + 'ms';
  };

  const formatBytes = (bytes: number): string => {
    if (bytes >= 1024 * 1024) {
      return (bytes / (1024 * 1024)).toFixed(1) + 'MB';
    }
    if (bytes >= 1024) {
      return (bytes / 1024).toFixed(1) + 'KB';
    }
    return bytes + 'B';
  };

  const getStatusIcon = (success: boolean) => {
    return success ? (
      <CheckCircle className="w-4 h-4 text-green-500" />
    ) : (
      <XCircle className="w-4 h-4 text-red-500" />
    );
  };

  const getDPITypeColor = (type: string): string => {
    const colors = {
      'Signature': 'text-blue-600 bg-blue-100',
      'Behavioral': 'text-purple-600 bg-purple-100',
      'MLBased': 'text-orange-600 bg-orange-100',
      'Hybrid': 'text-red-600 bg-red-100',
    };
    return colors[type as keyof typeof colors] || 'text-gray-600 bg-gray-100';
  };

  if (loading) {
    return (
      <div className={className}>
        <div className="animate-pulse space-y-3">
          {[...Array(compact ? 3 : 5)].map((_, i) => (
            <div key={i} className="h-16 bg-gray-200 rounded"></div>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={`p-4 bg-red-50 border border-red-200 rounded-lg ${className}`}>
        <p className="text-red-600 text-sm">{error}</p>
      </div>
    );
  }

  if (requests.length === 0) {
    return (
      <div className={`p-8 text-center text-gray-500 ${className}`}>
        <Clock className="w-12 h-12 mx-auto mb-4 text-gray-300" />
        <p>No requests data available</p>
      </div>
    );
  }

  if (compact) {
    return (
      <div className={`space-y-2 ${className}`}>
        {requests.slice(0, 5).map((request) => (
          <div key={request.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
            <div className="flex items-center space-x-3">
              {getStatusIcon(request.success)}
              <div>
                <p className="font-medium text-gray-900 text-sm">{request.domain}</p>
                <p className="text-xs text-gray-500">{formatLatency(request.latency)}</p>
              </div>
            </div>
            <div className="text-right">
              <span className={`px-2 py-1 rounded-full text-xs font-medium ${getDPITypeColor(request.dpi_type)}`}>
                {request.dpi_type}
              </span>
            </div>
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className={`overflow-x-auto ${className}`}>
      <table className="w-full">
        <thead>
          <tr className="border-b">
            <th className="text-left py-3 px-4 font-medium text-gray-900">Status</th>
            <th className="text-left py-3 px-4 font-medium text-gray-900">Domain</th>
            <th className="text-left py-3 px-4 font-medium text-gray-900">Method</th>
            <th className="text-left py-3 px-4 font-medium text-gray-900">Latency</th>
            <th className="text-left py-3 px-4 font-medium text-gray-900">DPI Type</th>
            <th className="text-left py-3 px-4 font-medium text-gray-900">Confidence</th>
            <th className="text-left py-3 px-4 font-medium text-gray-900">Data Transfer</th>
            <th className="text-left py-3 px-4 font-medium text-gray-900">Time</th>
            <th className="text-left py-3 px-4 font-medium text-gray-900">Actions</th>
          </tr>
        </thead>
        <tbody>
          {requests.map((request) => (
            <tr key={request.id} className="border-b hover:bg-gray-50">
              <td className="py-3 px-4">
                {getStatusIcon(request.success)}
              </td>
              <td className="py-3 px-4">
                <span className="font-medium text-gray-900">{request.domain}</span>
              </td>
              <td className="py-3 px-4">
                <span className="text-gray-600">{request.method}</span>
              </td>
              <td className="py-3 px-4">
                <span className="text-gray-600">{formatLatency(request.latency)}</span>
              </td>
              <td className="py-3 px-4">
                <span className={`px-2 py-1 rounded-full text-xs font-medium ${getDPITypeColor(request.dpi_type)}`}>
                  {request.dpi_type}
                </span>
              </td>
              <td className="py-3 px-4">
                <div className="flex items-center space-x-2">
                  <div className="w-full bg-gray-200 rounded-full h-2 max-w-16">
                    <div
                      className={`h-2 rounded-full ${
                        request.confidence >= 0.8 
                          ? 'bg-green-500' 
                          : request.confidence >= 0.6 
                          ? 'bg-yellow-500' 
                          : 'bg-red-500'
                      }`}
                      style={{ width: `${request.confidence * 100}%` }}
                    ></div>
                  </div>
                  <span className="text-sm text-gray-600">
                    {(request.confidence * 100).toFixed(0)}%
                  </span>
                </div>
              </td>
              <td className="py-3 px-4">
                <div className="text-sm text-gray-600">
                  <div>↓ {formatBytes(request.bytes_in)}</div>
                  <div>↑ {formatBytes(request.bytes_out)}</div>
                </div>
              </td>
              <td className="py-3 px-4">
                <span className="text-sm text-gray-500">
                  {new Date(request.created_at).toLocaleString()}
                </span>
              </td>
              <td className="py-3 px-4">
                <button
                  onClick={() => {
                    // TODO: Show request details modal
                    console.log('Show details for request:', request.request_id);
                  }}
                  className="p-1 hover:bg-gray-100 rounded"
                  title="View Details"
                >
                  <Eye className="w-4 h-4 text-gray-500" />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};
