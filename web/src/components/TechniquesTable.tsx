import React, { useState, useEffect } from 'react';
import { apiClient, type Technique } from '../api/client';
import { TrendingUp, TrendingDown, BarChart3 } from 'lucide-react';

interface TechniquesTableProps {
  limit?: number;
  compact?: boolean;
  className?: string;
}

export const TechniquesTable: React.FC<TechniquesTableProps> = ({ 
  limit, 
  compact = false, 
  className = '' 
}) => {
  const [techniques, setTechniques] = useState<Technique[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchTechniques = async () => {
      try {
        setLoading(true);
        const data = await apiClient.getTechniques();
        const filtered = limit ? data.slice(0, limit) : data;
        setTechniques(filtered);
        setError(null);
      } catch (err) {
        setError('Failed to fetch techniques');
        console.error('Error fetching techniques:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchTechniques();
  }, [limit]);

  const formatEffectiveness = (value: number): string => {
    return (value * 100).toFixed(1) + '%';
  };

  const getEffectivenessColor = (value: number): string => {
    if (value >= 0.8) return 'text-green-600 bg-green-100';
    if (value >= 0.6) return 'text-yellow-600 bg-yellow-100';
    return 'text-red-600 bg-red-100';
  };

  if (loading) {
    return (
      <div className={className}>
        <div className="animate-pulse space-y-3">
          {[...Array(compact ? 3 : 5)].map((_, i) => (
            <div key={i} className="h-12 bg-gray-200 rounded"></div>
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

  if (techniques.length === 0) {
    return (
      <div className={`p-8 text-center text-gray-500 ${className}`}>
        <BarChart3 className="w-12 h-12 mx-auto mb-4 text-gray-300" />
        <p>No techniques data available</p>
      </div>
    );
  }

  if (compact) {
    return (
      <div className={`space-y-2 ${className}`}>
        {techniques.map((technique, index) => (
          <div key={index} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
            <div className="flex items-center space-x-3">
              <div className={`px-2 py-1 rounded-full text-xs font-medium ${getEffectivenessColor(technique.effectiveness)}`}>
                {formatEffectiveness(technique.effectiveness)}
              </div>
              <span className="font-medium text-gray-900">{technique.technique}</span>
            </div>
            <div className="text-right">
              <p className="text-sm text-gray-500">{technique.sample_count} samples</p>
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
            <th className="text-left py-3 px-4 font-medium text-gray-900">Technique</th>
            <th className="text-left py-3 px-4 font-medium text-gray-900">Effectiveness</th>
            <th className="text-left py-3 px-4 font-medium text-gray-900">Sample Count</th>
            <th className="text-left py-3 px-4 font-medium text-gray-900">Domain</th>
            <th className="text-left py-3 px-4 font-medium text-gray-900">Last Update</th>
          </tr>
        </thead>
        <tbody>
          {techniques.map((technique, index) => (
            <tr key={index} className="border-b hover:bg-gray-50">
              <td className="py-3 px-4">
                <div className="flex items-center space-x-2">
                  <span className="font-medium text-gray-900">{technique.technique}</span>
                  {technique.effectiveness >= 0.8 && (
                    <TrendingUp className="w-4 h-4 text-green-500" />
                  )}
                  {technique.effectiveness < 0.5 && (
                    <TrendingDown className="w-4 h-4 text-red-500" />
                  )}
                </div>
              </td>
              <td className="py-3 px-4">
                <div className="flex items-center space-x-2">
                  <div className={`w-full bg-gray-200 rounded-full h-2 max-w-24`}>
                    <div
                      className={`h-2 rounded-full ${
                        technique.effectiveness >= 0.8 
                          ? 'bg-green-500' 
                          : technique.effectiveness >= 0.6 
                          ? 'bg-yellow-500' 
                          : 'bg-red-500'
                      }`}
                      style={{ width: `${technique.effectiveness * 100}%` }}
                    ></div>
                  </div>
                  <span className={`px-2 py-1 rounded-full text-xs font-medium ${getEffectivenessColor(technique.effectiveness)}`}>
                    {formatEffectiveness(technique.effectiveness)}
                  </span>
                </div>
              </td>
              <td className="py-3 px-4">
                <span className="text-gray-600">{technique.sample_count}</span>
              </td>
              <td className="py-3 px-4">
                <span className="text-gray-600">
                  {technique.domain || 'All domains'}
                </span>
              </td>
              <td className="py-3 px-4">
                <span className="text-sm text-gray-500">
                  {new Date(technique.last_update).toLocaleString()}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};
