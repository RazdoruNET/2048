import React, { useState } from 'react';
import { apiClient, type MLConfig, type FeedbackRequest } from '../api/client';
import { Settings, RefreshCw, Send } from 'lucide-react';

export const MLControls: React.FC = () => {
  const [config, setConfig] = useState<MLConfig>({
    learning_rate: 0.1,
    confidence_threshold: 0.85,
    model_update_interval: '30m',
    effectiveness_threshold: 0.7,
  });
  const [feedback, setFeedback] = useState<FeedbackRequest>({
    domain: '',
    technique: '',
    success: false,
  });
  const [loading, setLoading] = useState({
    config: false,
    feedback: false,
    retrain: false,
  });

  const handleConfigUpdate = async () => {
    try {
      setLoading(prev => ({ ...prev, config: true }));
      await apiClient.updateMLConfig(config);
      alert('ML configuration updated successfully!');
    } catch (error) {
      alert('Failed to update ML configuration');
      console.error('Config update error:', error);
    } finally {
      setLoading(prev => ({ ...prev, config: false }));
    }
  };

  const handleFeedbackSubmit = async () => {
    if (!feedback.domain || !feedback.technique) {
      alert('Please fill in all fields');
      return;
    }

    try {
      setLoading(prev => ({ ...prev, feedback: true }));
      await apiClient.postFeedback(feedback);
      alert('Feedback submitted successfully!');
      setFeedback({ domain: '', technique: '', success: false });
    } catch (error) {
      alert('Failed to submit feedback');
      console.error('Feedback error:', error);
    } finally {
      setLoading(prev => ({ ...prev, feedback: false }));
    }
  };

  const handleRetrain = async () => {
    if (!confirm('Are you sure you want to retrain the ML model? This may take several minutes.')) {
      return;
    }

    try {
      setLoading(prev => ({ ...prev, retrain: true }));
      await apiClient.retrainModel(true);
      alert('Model retraining started!');
    } catch (error) {
      alert('Failed to start model retraining');
      console.error('Retrain error:', error);
    } finally {
      setLoading(prev => ({ ...prev, retrain: false }));
    }
  };

  return (
    <div className="space-y-6">
      {/* ML Configuration */}
      <div className="bg-white rounded-lg shadow-sm border p-6">
        <div className="flex items-center space-x-2 mb-4">
          <Settings className="w-5 h-5 text-gray-600" />
          <h3 className="text-lg font-semibold text-gray-900">ML Configuration</h3>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Learning Rate
            </label>
            <input
              type="number"
              step="0.01"
              min="0"
              max="1"
              value={config.learning_rate}
              onChange={(e) => setConfig(prev => ({ 
                ...prev, 
                learning_rate: parseFloat(e.target.value) 
              }))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Confidence Threshold
            </label>
            <input
              type="number"
              step="0.01"
              min="0"
              max="1"
              value={config.confidence_threshold}
              onChange={(e) => setConfig(prev => ({ 
                ...prev, 
                confidence_threshold: parseFloat(e.target.value) 
              }))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Model Update Interval
            </label>
            <select
              value={config.model_update_interval}
              onChange={(e) => setConfig(prev => ({ 
                ...prev, 
                model_update_interval: e.target.value 
              }))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="5m">5 minutes</option>
              <option value="15m">15 minutes</option>
              <option value="30m">30 minutes</option>
              <option value="1h">1 hour</option>
              <option value="6h">6 hours</option>
              <option value="24h">24 hours</option>
            </select>
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Effectiveness Threshold
            </label>
            <input
              type="number"
              step="0.01"
              min="0"
              max="1"
              value={config.effectiveness_threshold}
              onChange={(e) => setConfig(prev => ({ 
                ...prev, 
                effectiveness_threshold: parseFloat(e.target.value) 
              }))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>
        
        <div className="mt-4">
          <button
            onClick={handleConfigUpdate}
            disabled={loading.config}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
          >
            <Settings className="w-4 h-4" />
            <span>{loading.config ? 'Updating...' : 'Update Configuration'}</span>
          </button>
        </div>
      </div>

      {/* Model Retraining */}
      <div className="bg-white rounded-lg shadow-sm border p-6">
        <div className="flex items-center space-x-2 mb-4">
          <RefreshCw className="w-5 h-5 text-gray-600" />
          <h3 className="text-lg font-semibold text-gray-900">Model Retraining</h3>
        </div>
        
        <p className="text-gray-600 mb-4">
          Retrain the ML model with the latest data. This process may take several minutes and 
          temporarily affect performance.
        </p>
        
        <button
          onClick={handleRetrain}
          disabled={loading.retrain}
          className="px-4 py-2 bg-orange-600 text-white rounded-md hover:bg-orange-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
        >
          <RefreshCw className={`w-4 h-4 ${loading.retrain ? 'animate-spin' : ''}`} />
          <span>{loading.retrain ? 'Retraining...' : 'Start Retraining'}</span>
        </button>
      </div>

      {/* Feedback Submission */}
      <div className="bg-white rounded-lg shadow-sm border p-6">
        <div className="flex items-center space-x-2 mb-4">
          <Send className="w-5 h-5 text-gray-600" />
          <h3 className="text-lg font-semibold text-gray-900">Submit Feedback</h3>
        </div>
        
        <p className="text-gray-600 mb-4">
          Help improve the ML model by providing feedback on technique effectiveness.
        </p>
        
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Domain
            </label>
            <input
              type="text"
              value={feedback.domain}
              onChange={(e) => setFeedback(prev => ({ ...prev, domain: e.target.value }))}
              placeholder="example.com"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Technique
            </label>
            <select
              value={feedback.technique}
              onChange={(e) => setFeedback(prev => ({ ...prev, technique: e.target.value }))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">Select technique...</option>
              <option value="fragmentation">Fragmentation</option>
              <option value="encryption">Encryption</option>
              <option value="headers">Headers</option>
              <option value="protocol_mask">Protocol Mask</option>
              <option value="adaptive_frag">Adaptive Fragmentation</option>
              <option value="timing_random">Timing Randomization</option>
            </select>
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Result
            </label>
            <select
              value={feedback.success ? 'true' : 'false'}
              onChange={(e) => setFeedback(prev => ({ 
                ...prev, 
                success: e.target.value === 'true' 
              }))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="true">Success</option>
              <option value="false">Failure</option>
            </select>
          </div>
        </div>
        
        <button
          onClick={handleFeedbackSubmit}
          disabled={loading.feedback}
          className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
        >
          <Send className="w-4 h-4" />
          <span>{loading.feedback ? 'Submitting...' : 'Submit Feedback'}</span>
        </button>
      </div>
    </div>
  );
};
