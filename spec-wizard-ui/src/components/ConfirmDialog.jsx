import React from 'react';
import { X, AlertTriangle, CheckCircle2 } from 'lucide-react';

const ConfirmDialog = ({ 
  isOpen, 
  onClose, 
  onConfirm, 
  title, 
  message, 
  confirmText = "Confirmar", 
  cancelText = "Cancelar",
  type = "warning",
  isLoading = false
}) => {
  if (!isOpen) return null;

  const themes = {
    warning: {
      icon: <AlertTriangle className="w-6 h-6 text-amber-400" />,
      bg: "bg-amber-500/10",
      border: "border-amber-500/20",
      button: "bg-amber-600 hover:bg-amber-700 shadow-amber-900/20"
    },
    danger: {
      icon: <AlertTriangle className="w-6 h-6 text-red-400" />,
      bg: "bg-red-500/10",
      border: "border-red-500/20",
      button: "bg-red-600 hover:bg-red-700 shadow-red-900/20"
    },
    success: {
      icon: <CheckCircle2 className="w-6 h-6 text-emerald-400" />,
      bg: "bg-emerald-500/10",
      border: "border-emerald-500/20",
      button: "bg-emerald-600 hover:bg-emerald-700 shadow-emerald-900/20"
    }
  };

  const theme = themes[type] || themes.warning;

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4">
      <div 
        className="absolute inset-0 bg-black/60 backdrop-blur-sm animate-in fade-in duration-200" 
        onClick={onClose} 
      />
      
      <div className="relative w-full max-w-md bg-slate-900 border border-white/10 rounded-3xl shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200">
        <div className="p-6">
          <div className="flex items-start gap-4">
            <div className={`p-3 rounded-2xl ${theme.bg} ${theme.border} border`}>
              {theme.icon}
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
            disabled={isLoading}
            className="flex-1 px-4 py-3 rounded-2xl bg-white/5 hover:bg-white/10 text-white font-bold text-sm transition-all border border-white/10 disabled:opacity-50"
          >
            {cancelText}
          </button>
          <button
            onClick={onConfirm}
            disabled={isLoading}
            className={`flex-1 px-4 py-3 rounded-2xl text-white font-bold text-sm transition-all shadow-lg flex items-center justify-center gap-2 disabled:opacity-50 ${theme.button}`}
          >
            {isLoading && <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />}
            {confirmText}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ConfirmDialog;
