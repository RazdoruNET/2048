import React, { useState, useEffect } from 'react';
import { X, Clock, Globe, Activity, AlertCircle, CheckCircle, XCircle } from 'lucide-react';

interface RequestDetailsModalProps {
  isOpen: boolean;
  onClose: () => void;
  requestId: string;
}

interface RequestDetails {
  request: {
    id: number;
    request_id: string;
    domain: string;
    method: string;
    status_code: number;
    success: boolean;
    techniques: string;
    timestamp: string;
  };
  events: any[];
  metadata: {
    request_id: string;
    dpi_type: string;
    confidence: number;
  };
  timing: {
    created_at: string;
    latency_ms: number;
    bytes_in: number;
    bytes_out: number;
  };
  stats: {
    total_events: number;
    dpi_detected: boolean;
  };
}

const RequestDetailsModal: React.FC<RequestDetailsModalProps> = ({ 
  isOpen, 
  onClose, 
  requestId 
}) => {
  const [details, setDetails] = useState<RequestDetails | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen && requestId) {
      fetchRequestDetails();
    }
  }, [isOpen, requestId]);

  const fetchRequestDetails = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/v1/requests/${requestId}/modal`);
      if (!response.ok) {
        throw new Error('Failed to fetch request details');
      }
      const data = await response.json();
      setDetails(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatDuration = (ms: number): string => {
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  };

  const getStatusCodeColor = (statusCode: number): string => {
    if (statusCode >= 200 && statusCode < 300) return 'text-green-600';
    if (statusCode >= 300 && statusCode < 400) return 'text-yellow-600';
    if (statusCode >= 400 && statusCode < 500) return 'text-orange-600';
    if (statusCode >= 500) return 'text-red-600';
    return 'text-gray-600';
  };

  const getSuccessIcon = (success: boolean) => {
    if (success) {
      return <CheckCircle className="w-5 h-5 text-green-500" />;
    } else {
      return <XCircle className="w-5 h-5 text-red-500" />;
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b">
          <h2 className="text-xl font-semibold text-gray-900">Request Details</h2>
          <button
            onClick={onClose}
            className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
          >
            <X className="w-5 h-5 text-gray-500" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 overflow-y-auto max-h-[calc(90vh-8rem)]">
          {loading && (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            </div>
          )}

          {error && (
            <div className="flex items-center gap-2 p-4 bg-red-50 border border-red-200 rounded-lg">
              <AlertCircle className="w-5 h-5 text-red-500" />
              <span className="text-red-700">{error}</span>
            </div>
          )}

          {details && !loading && !error && (
            <div className="space-y-6">
              {/* Request Overview */}
              <div className="bg-gray-50 rounded-lg p-4">
                <h3 className="text-lg font-medium text-gray-900 mb-4">Request Overview</h3>
                <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
                  <div>
                    <label className="text-sm font-medium text-gray-500">Request ID</label>
                    <p className="text-sm text-gray-900 font-mono">{details.request.request_id}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-500">Domain</label>
                    <p className="text-sm text-gray-900 flex items-center gap-1">
                      <Globe className="w-4 h-4" />
                      {details.request.domain}
                    </p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-500">Method</label>
                    <p className="text-sm text-gray-900 font-mono">{details.request.method}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-500">Status Code</label>
                    <p className={`text-sm font-bold ${getStatusCodeColor(details.request.status_code)}`}>
                      {details.request.status_code}
                    </p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-500">Success</label>
                    <div className="flex items-center gap-2">
                      {getSuccessIcon(details.request.success)}
                      <span className="text-sm text-gray-900">
                        {details.request.success ? 'Success' : 'Failed'}
                      </span>
                    </div>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-500">Techniques</label>
                    <p className="text-sm text-gray-900">{details.request.techniques || 'None'}</p>
                  </div>
                </div>
              </div>

              {/* Timing Information */}
              <div className="bg-blue-50 rounded-lg p-4">
                <h3 className="text-lg font-medium text-gray-900 mb-4 flex items-center gap-2">
                  <Clock className="w-5 h-5" />
                  Timing Information
                </h3>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div>
                    <label className="text-sm font-medium text-gray-500">Created At</label>
                    <p className="text-sm text-gray-900">
                      {new Date(details.timing.created_at).toLocaleString()}
                    </p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-500">Latency</label>
                    <p className="text-sm text-gray-900">{formatDuration(details.timing.latency_ms)}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-500">Bytes In</label>
                    <p className="text-sm text-gray-900">{formatBytes(details.timing.bytes_in)}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-500">Bytes Out</label>
                    <p className="text-sm text-gray-900">{formatBytes(details.timing.bytes_out)}</p>
                  </div>
                </div>
              </div>

              {/* DPI Detection */}
              {details.stats.dpi_detected && (
                <div className="bg-yellow-50 rounded-lg p-4">
                  <h3 className="text-lg font-medium text-gray-900 mb-4 flex items-center gap-2">
                    <Activity className="w-5 h-5" />
                    DPI Detection
                  </h3>
                  <div className="space-y-2">
                    <div className="flex justify-between">
                      <span className="text-sm font-medium text-gray-500">DPI Type</span>
                      <span className="text-sm text-gray-900">{details.metadata.dpi_type}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-sm font-medium text-gray-500">Confidence</span>
                      <span className="text-sm text-gray-900">
                        {(details.metadata.confidence * 100).toFixed(1)}%
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-sm font-medium text-gray-500">Events</span>
                      <span className="text-sm text-gray-900">{details.stats.total_events}</span>
                    </div>
                  </div>
                </div>
              )}

              {/* Events */}
              {details.events && details.events.length > 0 && (
                <div className="bg-gray-50 rounded-lg p-4">
                  <h3 className="text-lg font-medium text-gray-900 mb-4">Events</h3>
                  <div className="space-y-2">
                    {details.events.map((event, index) => (
                      <div key={index} className="bg-white p-3 rounded border border-gray-200">
                        <div className="flex justify-between items-start">
                          <div>
                            <p className="text-sm font-medium text-gray-900">{event.type}</p>
                            <p className="text-sm text-gray-600">{event.description}</p>
                          </div>
                          <span className="text-xs text-gray-500">
                            {new Date(event.timestamp).toLocaleTimeString()}
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Statistics */}
              <div className="bg-green-50 rounded-lg p-4">
                <h3 className="text-lg font-medium text-gray-900 mb-4">Statistics</h3>
                <div className="grid grid-cols-2 gap-4">
                  <div className="text-center">
                    <div className="text-2xl font-bold text-green-600">
                      {details.stats.total_events}
                    </div>
                    <div className="text-sm text-gray-600">Total Events</div>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-blue-600">
                      {details.stats.dpi_detected ? 'Yes' : 'No'}
                    </div>
                    <div className="text-sm text-gray-600">DPI Detected</div>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-end p-6 border-t bg-gray-50">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};

export default RequestDetailsModal;
