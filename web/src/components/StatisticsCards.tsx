import React from 'react';
import { TrendingUp, Activity, Zap, AlertCircle } from 'lucide-react';
import { type MLStatistics } from '../api/client';

interface StatisticsCardsProps {
  statistics: MLStatistics;
  className?: string;
}

export const StatisticsCards: React.FC<StatisticsCardsProps> = ({ 
  statistics, 
  className = '' 
}) => {
  const formatNumber = (num: number): string => {
    if (num >= 1000000) {
      return (num / 1000000).toFixed(1) + 'M';
    }
    if (num >= 1000) {
      return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
  };

  const formatLatency = (ms: number): string => {
    if (ms >= 1000) {
      return (ms / 1000).toFixed(1) + 's';
    }
    return ms + 'ms';
  };

  const formatPercentage = (value: number): string => {
    return (value * 100).toFixed(1) + '%';
  };

  const stats = [
    {
      title: 'Total Requests',
      value: formatNumber(statistics.total_requests),
      icon: Activity,
      color: 'blue',
      trend: null,
    },
    {
      title: 'Success Rate',
      value: formatPercentage(statistics.success_rate),
      icon: TrendingUp,
      color: 'green',
      trend: statistics.success_rate > 0.8 ? 'up' : 'down',
    },
    {
      title: 'Avg Latency',
      value: formatLatency(statistics.average_latency),
      icon: Zap,
      color: 'yellow',
      trend: statistics.average_latency < 200 ? 'up' : 'down',
    },
    {
      title: 'Error Rate',
      value: formatPercentage(statistics.error_rate),
      icon: AlertCircle,
      color: 'red',
      trend: statistics.error_rate < 0.1 ? 'up' : 'down',
    },
  ];

  const getColorClasses = (color: string) => {
    const colors = {
      blue: {
        bg: 'bg-blue-100',
        text: 'text-blue-600',
      },
      green: {
        bg: 'bg-green-100',
        text: 'text-green-600',
      },
      yellow: {
        bg: 'bg-yellow-100',
        text: 'text-yellow-600',
      },
      red: {
        bg: 'bg-red-100',
        text: 'text-red-600',
      },
    };
    return colors[color as keyof typeof colors] || colors.blue;
  };

  return (
    <div className={`p-6 bg-white rounded-lg shadow-sm border ${className}`}>
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Performance Metrics</h3>
      
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {stats.map((stat, index) => {
          const Icon = stat.icon;
          const colors = getColorClasses(stat.color);
          
          return (
            <div key={index} className="text-center">
              <div className={`inline-flex p-3 rounded-lg ${colors.bg} mb-3`}>
                <Icon className={`w-6 h-6 ${colors.text}`} />
              </div>
              
              <div className="space-y-1">
                <p className="text-2xl font-bold text-gray-900">{stat.value}</p>
                <p className="text-sm text-gray-500">{stat.title}</p>
              </div>
              
              {stat.trend && (
                <div className="mt-2">
                  <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                    stat.trend === 'up' 
                      ? 'bg-green-100 text-green-800' 
                      : 'bg-red-100 text-red-800'
                  }`}>
                    {stat.trend === 'up' ? '↑' : '↓'}
                  </span>
                </div>
              )}
            </div>
          );
        })}
      </div>

      <div className="mt-6 pt-4 border-t grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="text-center">
          <p className="text-sm text-gray-500">Throughput</p>
          <p className="text-lg font-semibold text-gray-900">
            {formatNumber(statistics.throughput_bps)} bps
          </p>
        </div>
        <div className="text-center">
          <p className="text-sm text-gray-500">Active Connections</p>
          <p className="text-lg font-semibold text-gray-900">
            {statistics.active_connections}
          </p>
        </div>
      </div>
    </div>
  );
};
