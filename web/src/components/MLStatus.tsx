import React, { useState, useEffect } from 'react';
import { apiClient, type MLStatus } from '../api/client';
import { wsClient, type MLStatusUpdate } from '../api/websocket';
import { Activity, Cpu, Zap } from 'lucide-react';

interface MLStatusProps {
  className?: string;
}

export const MLStatusCard: React.FC<MLStatusProps> = ({ className = '' }) => {
  const [status, setStatus] = useState<MLStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        setLoading(true);
        const data = await apiClient.getMLStatus();
        setStatus(data);
        setError(null);
      } catch (err) {
        setError('Failed to fetch ML status');
        console.error('Error fetching ML status:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchStatus();

    // Set up WebSocket for real-time updates
    wsClient.onMLStatusUpdate((update: MLStatusUpdate) => {
      setStatus({
        enabled: update.enabled,
        last_update: update.last_update,
        model_version: update.model_version,
        techniques: status?.techniques || [],
      });
    });

    // Connect WebSocket if not already connected
    if (!wsClient.isConnected()) {
      wsClient.connect().catch(console.error);
    }

    return () => {
      wsClient.disconnect();
    };
  }, []);

  if (loading) {
    return (
      <div className={`p-6 bg-white rounded-lg shadow-sm border ${className}`}>
        <div className="animate-pulse">
          <div className="h-4 bg-gray-200 rounded w-1/4 mb-4"></div>
          <div className="space-y-2">
            <div className="h-3 bg-gray-200 rounded"></div>
            <div className="h-3 bg-gray-200 rounded w-3/4"></div>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={`p-6 bg-red-50 border border-red-200 rounded-lg ${className}`}>
        <div className="flex items-center space-x-2 text-red-800">
          <Activity className="w-5 h-5" />
          <span className="font-medium">Error</span>
        </div>
        <p className="text-red-600 text-sm mt-2">{error}</p>
      </div>
    );
  }

  if (!status) {
    return null;
  }

  return (
    <div className={`p-6 bg-white rounded-lg shadow-sm border ${className}`}>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-900">ML Engine Status</h3>
        <div className={`px-3 py-1 rounded-full text-xs font-medium ${
          status.enabled 
            ? 'bg-green-100 text-green-800' 
            : 'bg-gray-100 text-gray-800'
        }`}>
          {status.enabled ? 'Active' : 'Inactive'}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-blue-100 rounded-lg">
            <Cpu className="w-5 h-5 text-blue-600" />
          </div>
          <div>
            <p className="text-sm text-gray-500">Model Version</p>
            <p className="font-medium text-gray-900">{status.model_version}</p>
          </div>
        </div>

        <div className="flex items-center space-x-3">
          <div className="p-2 bg-green-100 rounded-lg">
            <Zap className="w-5 h-5 text-green-600" />
          </div>
          <div>
            <p className="text-sm text-gray-500">Techniques</p>
            <p className="font-medium text-gray-900">{status.techniques.length}</p>
          </div>
        </div>

        <div className="flex items-center space-x-3">
          <div className="p-2 bg-purple-100 rounded-lg">
            <Activity className="w-5 h-5 text-purple-600" />
          </div>
          <div>
            <p className="text-sm text-gray-500">Last Update</p>
            <p className="font-medium text-gray-900">
              {new Date(status.last_update).toLocaleTimeString()}
            </p>
          </div>
        </div>
      </div>

      {status.techniques.length > 0 && (
        <div className="mt-4 pt-4 border-t">
          <p className="text-sm text-gray-500 mb-2">Active Techniques:</p>
          <div className="flex flex-wrap gap-2">
            {status.techniques.map((technique, index) => (
              <span
                key={index}
                className="px-2 py-1 bg-gray-100 text-gray-700 text-xs rounded-full"
              >
                {technique}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
