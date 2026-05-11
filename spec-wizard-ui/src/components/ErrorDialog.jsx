import React from 'react';
import { X, AlertCircle } from 'lucide-react';

const ErrorDialog = ({ 
  isOpen, 
  onClose, 
  title = "Erro", 
  message 
}) => {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[110] flex items-center justify-center p-4">
      <div 
        className="absolute inset-0 bg-black/60 backdrop-blur-sm animate-in fade-in duration-200" 
        onClick={onClose} 
      />
      
      <div className="relative w-full max-w-md bg-slate-900 border border-red-500/30 rounded-3xl shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200">
        <div className="p-6">
          <div className="flex items-start gap-4">
            <div className="p-3 rounded-2xl bg-red-500/10 border border-red-500/20">
              <AlertCircle className="w-6 h-6 text-red-400" />
            </div>
            <div className="flex-1">
              <h3 className="text-xl font-bold text-white mb-2">{title}</h3>
              <p className="text-slate-400 text-sm leading-relaxed">
                {message}
              </p>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-3 p-4 bg-white/5 border-t border-white/10">
          <button
            onClick={onClose}
            className="flex-1 px-4 py-3 rounded-2xl bg-red-600 hover:bg-red-700 text-white font-bold text-sm transition-all shadow-lg shadow-red-900/20 border border-red-500/30"
          >
            Entendido
          </button>
        </div>
      </div>
    </div>
  );
};

export default ErrorDialog;
