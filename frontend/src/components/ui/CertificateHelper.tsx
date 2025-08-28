import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { AlertTriangle, ExternalLink, X, CheckCircle } from 'lucide-react';

interface CertificateHelperProps {
  isVisible: boolean;
  onClose: () => void;
  onRetry: () => void;
}

export const CertificateHelper: React.FC<CertificateHelperProps> = ({
  isVisible,
  onClose,
  onRetry
}) => {
  const [step, setStep] = useState<'instructions' | 'testing'>('instructions');

  const handleAcceptCertificate = () => {
    // Open backend URL in new tab for certificate acceptance
    const backendHost = window.location.hostname === 'localhost' ? 'localhost' : 'spidexd.local';
    window.open(`https://${backendHost}:8000/health`, '_blank');
    setStep('testing');
  };

  const handleRetryConnection = () => {
    setStep('instructions');
    onRetry();
    onClose();
  };

  return (
    <AnimatePresence>
      {isVisible && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4"
        >
          <motion.div
            initial={{ scale: 0.9, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            exit={{ scale: 0.9, opacity: 0 }}
            className="bg-app-secondary/95 backdrop-blur-xl rounded-2xl border border-app-primary/30 p-6 max-w-md w-full shadow-2xl"
          >
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center">
                <AlertTriangle className="w-6 h-6 text-yellow-400 mr-2" />
                <h3 className="text-lg font-bold text-app-primary">Certificate Setup</h3>
              </div>
              <button
                onClick={onClose}
                className="p-1 rounded-lg hover:bg-app-primary/10 transition-colors"
              >
                <X className="w-5 h-5 text-app-tertiary" />
              </button>
            </div>

            {step === 'instructions' && (
              <div className="space-y-4">
                <p className="text-app-tertiary text-sm">
                  The backend uses a self-signed certificate for HTTPS. You need to accept it in your browser first.
                </p>

                <div className="bg-app-primary/10 rounded-lg p-4 space-y-3">
                  <h4 className="font-semibold text-app-secondary text-sm">Steps:</h4>
                  <ol className="text-xs text-app-tertiary space-y-2 list-decimal list-inside">
                    <li>Click "Accept Certificate" below</li>
                    <li>A new tab will open to the backend API</li>
                    <li>Click "Advanced" → "Proceed to {window.location.hostname} (unsafe)"</li>
                    <li>You should see a health status response</li>
                    <li>Close the tab and click "Retry Connection"</li>
                  </ol>
                </div>

                <div className="flex space-x-3">
                  <button
                    onClick={handleAcceptCertificate}
                    className="flex-1 flex items-center justify-center px-4 py-2 bg-app-tertiary/30 hover:bg-app-tertiary/40 text-app-primary rounded-lg transition-colors text-sm font-medium"
                  >
                    <ExternalLink className="w-4 h-4 mr-2" />
                    Accept Certificate
                  </button>
                </div>
              </div>
            )}

            {step === 'testing' && (
              <div className="space-y-4 text-center">
                <CheckCircle className="w-12 h-12 text-green-400 mx-auto" />
                <div>
                  <h4 className="font-semibold text-app-secondary mb-2">Certificate Accepted?</h4>
                  <p className="text-app-tertiary text-sm mb-4">
                    If you've accepted the certificate in the other tab, click retry to test the connection.
                  </p>
                </div>

                <div className="flex space-x-3">
                  <button
                    onClick={() => setStep('instructions')}
                    className="flex-1 px-4 py-2 border border-app-primary/30 text-app-tertiary rounded-lg hover:bg-app-primary/10 transition-colors text-sm"
                  >
                    Back
                  </button>
                  <button
                    onClick={handleRetryConnection}
                    className="flex-1 px-4 py-2 bg-green-500/20 hover:bg-green-500/30 text-green-400 rounded-lg transition-colors text-sm font-medium"
                  >
                    Retry Connection
                  </button>
                </div>
              </div>
            )}
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
};
