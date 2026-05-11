import React, { useState, useEffect } from 'react';
import { MLStatusCard } from './MLStatus';
import { StatisticsCards } from './StatisticsCards';
import { TechniquesTable } from './TechniquesTable';
import { RequestsList } from './RequestsList';
import { MLControls } from './MLControls';
import { apiClient, type MLStatistics } from '../api/client';
import { BarChart3, Settings, Activity } from 'lucide-react';

export const Dashboard: React.FC = () => {
  const [statistics, setStatistics] = useState<MLStatistics | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'overview' | 'requests' | 'techniques' | 'controls'>('overview');

  useEffect(() => {
    const fetchStatistics = async () => {
      try {
        const data = await apiClient.getMLStatistics();
        setStatistics(data);
      } catch (error) {
        console.error('Failed to fetch statistics:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchStatistics();
    const interval = setInterval(fetchStatistics, 30000); // Update every 30 seconds

    return () => clearInterval(interval);
  }, []);

  const tabs = [
    { id: 'overview', label: 'Overview', icon: BarChart3 },
    { id: 'requests', label: 'Requests', icon: Activity },
    { id: 'techniques', label: 'Techniques', icon: Settings },
    { id: 'controls', label: 'ML Controls', icon: Settings },
  ] as const;

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 p-6">
        <div className="max-w-7xl mx-auto">
          <div className="animate-pulse">
            <div className="h-8 bg-gray-200 rounded w-1/4 mb-8"></div>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
              {[...Array(4)].map((_, i) => (
                <div key={i} className="h-32 bg-gray-200 rounded-lg"></div>
              ))}
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">ML Monitoring Dashboard</h1>
              <p className="text-sm text-gray-500 mt-1">
                Real-time monitoring and control of ML DPI bypass system
              </p>
            </div>
            <div className="flex items-center space-x-2">
              <div className={`px-3 py-1 rounded-full text-xs font-medium ${
                statistics ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
              }`}>
                {statistics ? 'Connected' : 'Disconnected'}
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation Tabs */}
      <div className="bg-white border-b">
        <div className="max-w-7xl mx-auto px-6">
          <nav className="flex space-x-8">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex items-center space-x-2 py-4 px-1 border-b-2 font-medium text-sm ${
                    activeTab === tab.id
                      ? 'border-blue-500 text-blue-600'
                      : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                  }`}
                >
                  <Icon className="w-4 h-4" />
                  <span>{tab.label}</span>
                </button>
              );
            })}
          </nav>
        </div>
      </div>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-6 py-8">
        {/* Overview Tab */}
        {activeTab === 'overview' && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <MLStatusCard />
              {statistics && <StatisticsCards statistics={statistics} />}
            </div>
            
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <div className="bg-white rounded-lg shadow-sm border p-6">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">Recent Activity</h3>
                <RequestsList limit={10} compact />
              </div>
              <div className="bg-white rounded-lg shadow-sm border p-6">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">Top Techniques</h3>
                <TechniquesTable limit={5} compact />
              </div>
            </div>
          </div>
        )}

        {/* Requests Tab */}
        {activeTab === 'requests' && (
          <div className="bg-white rounded-lg shadow-sm border">
            <div className="p-6 border-b">
              <h2 className="text-lg font-semibold text-gray-900">Request Tracing</h2>
              <p className="text-sm text-gray-500 mt-1">
                Detailed view of all processed requests and their DPI analysis
              </p>
            </div>
            <div className="p-6">
              <RequestsList />
            </div>
          </div>
        )}

        {/* Techniques Tab */}
        {activeTab === 'techniques' && (
          <div className="bg-white rounded-lg shadow-sm border">
            <div className="p-6 border-b">
              <h2 className="text-lg font-semibold text-gray-900">Technique Effectiveness</h2>
              <p className="text-sm text-gray-500 mt-1">
                Performance metrics for DPI bypass techniques
              </p>
            </div>
            <div className="p-6">
              <TechniquesTable />
            </div>
          </div>
        )}

        {/* Controls Tab */}
        {activeTab === 'controls' && (
          <div className="space-y-6">
            <MLControls />
          </div>
        )}
      </main>
    </div>
  );
};
