import React, { useState, useEffect } from 'react';
import { translations } from '../translations';
import { 
  X, 
  Save, 
  FileText, 
  Info,
  Loader2,
  CheckCircle2,
  AlertCircle
} from 'lucide-react';
import ConfirmDialog from './ConfirmDialog';
import ErrorDialog from './ErrorDialog';

const TaskSpecModal = ({ isOpen, onClose, taskId, sprintId, projectId, language }) => {
  const t = translations[language] || translations.en;
  const [spec, setSpec] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [status, setStatus] = useState(null); // 'success', 'error'
  const [isBootstrapping, setIsBootstrapping] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [errorInfo, setErrorInfo] = useState({ isOpen: false, message: '' });

  useEffect(() => {
    if (isOpen && taskId && projectId) {
      fetchSpec();
    }
  }, [isOpen, taskId, projectId]);

  const fetchSpec = async () => {
    setIsLoading(true);
    try {
      const response = await fetch(`/api/project/task-spec?path=${encodeURIComponent(projectId)}&task_id=${taskId}`);
      if (response.ok) {
        const data = await response.json();
        setSpec(data.content || '');
      }
    } catch (error) {
      console.error('Error fetching task spec:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSave = async () => {
    setIsSaving(true);
    setStatus(null);
    try {
      const response = await fetch('/api/project/task-spec', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          path: projectId,
          task_id: taskId,
          content: spec
        }),
      });

      if (response.ok) {
        setStatus('success');
        setTimeout(() => setStatus(null), 3000);
      } else {
        setStatus('error');
      }
    } catch (error) {
      console.error('Error saving task spec:', error);
      setStatus('error');
    } finally {
      setIsSaving(false);
    }
  };

  const handleBootstrap = async () => {
    setShowConfirm(false);
    setIsBootstrapping(true);
    setStatus(null);
    try {
      const response = await fetch('/api/project/task-spec/bootstrap', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          path: projectId,
          sprint_id: String(sprintId),
          task_id: String(taskId)
        }),
      });

      if (response.ok) {
        const data = await response.json();
        setSpec(data.content || '');
        setStatus('success');
        setTimeout(() => setStatus(null), 3000);
      } else {
        const errorData = await response.json();
        setErrorInfo({ isOpen: true, message: errorData.error || 'Erro ao gerar spec' });
        setStatus('error');
      }
    } catch (error) {
      console.error('Error bootstrapping task spec:', error);
      setErrorInfo({ isOpen: true, message: 'Falha na conexão com o servidor.' });
      setStatus('error');
    } finally {
      setIsBootstrapping(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-in fade-in duration-200">
      <div className="bg-[#1a1b1e] border border-white/10 rounded-xl w-full max-w-4xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200">
        
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-white/10 bg-white/5">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-indigo-500/20 rounded-lg">
              <FileText className="w-5 h-5 text-indigo-400" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-white">{t.task_spec}</h2>
              <p className="text-xs text-white/40">Task ID: {taskId} | Sprint: {sprintId}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowConfirm(true)}
              disabled={isBootstrapping || isLoading}
              className="flex items-center gap-2 px-3 py-1.5 bg-amber-500/10 hover:bg-amber-500/20 text-amber-400 border border-amber-500/20 rounded-lg text-xs font-bold transition-all disabled:opacity-50"
              title={t.generate_with_architect}
            >
              {isBootstrapping ? (
                <Loader2 className="w-3 h-3 animate-spin" />
              ) : (
                <CheckCircle2 className="w-3 h-3" />
              )}
              {t.bootstrap_spec}
            </button>
            <button 
              onClick={onClose}
              className="p-2 hover:bg-white/10 rounded-full transition-colors text-white/60 hover:text-white"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-4">
          <div className="flex items-start gap-3 p-4 bg-blue-500/10 border border-blue-500/20 rounded-lg">
            <Info className="w-5 h-5 text-blue-400 mt-0.5 flex-shrink-0" />
            <p className="text-sm text-blue-100/80 leading-relaxed">
              {t.task_spec_help}
            </p>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium text-white/70 block px-1">
              {t.task_spec_desc}
            </label>
            <div className="relative group">
              {isLoading ? (
                <div className="absolute inset-0 flex items-center justify-center bg-black/20 backdrop-blur-[1px] rounded-lg z-10">
                  <Loader2 className="w-8 h-8 text-indigo-500 animate-spin" />
                </div>
              ) : null}
              <textarea
                value={spec}
                onChange={(e) => setSpec(e.target.value)}
                placeholder="# Implementation Details\n\n- Use the X design pattern...\n- Ensure that Y is validated..."
                className="w-full h-[400px] bg-black/40 border border-white/10 rounded-lg p-4 text-white font-mono text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500/50 transition-all resize-none placeholder:text-white/10"
              />
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-white/10 bg-white/5 flex items-center justify-between">
          <div className="flex items-center gap-2">
            {status === 'success' && (
              <div className="flex items-center gap-2 text-emerald-400 text-sm animate-in slide-in-from-left-2">
                <CheckCircle2 className="w-4 h-4" />
                <span>{t.spec_saved || 'Saved!'}</span>
              </div>
            )}
            {status === 'error' && (
              <div className="flex items-center gap-2 text-rose-400 text-sm animate-in slide-in-from-left-2">
                <AlertCircle className="w-4 h-4" />
                <span>{t.error_loading || 'Error!'}</span>
              </div>
            )}
          </div>
          
          <div className="flex items-center gap-3">
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-white/60 hover:text-white hover:bg-white/5 rounded-lg transition-all"
            >
              {t.cancel}
            </button>
            <button
              onClick={handleSave}
              disabled={isSaving || isLoading}
              className={`
                flex items-center gap-2 px-6 py-2 rounded-lg text-sm font-bold transition-all
                ${isSaving 
                  ? 'bg-indigo-500/50 cursor-not-allowed text-white/50' 
                  : 'bg-indigo-600 hover:bg-indigo-500 text-white shadow-lg shadow-indigo-500/20 active:scale-95'
                }
              `}
            >
              {isSaving ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Save className="w-4 h-4" />
              )}
              {isSaving ? t.saving : t.save_changes}
            </button>
          </div>
        </div>
      </div>

      <ConfirmDialog
        isOpen={showConfirm}
        onClose={() => setShowConfirm(false)}
        onConfirm={handleBootstrap}
        title={t.bootstrap_spec}
        message={t.confirm_bootstrap}
        confirmText={t.generate_with_architect || "Gerar"}
        cancelText={t.discard}
        isLoading={isBootstrapping}
        type="warning"
      />

      <ErrorDialog
        isOpen={errorInfo.isOpen}
        onClose={() => setErrorInfo({ isOpen: false, message: '' })}
        title={t.error || "Erro"}
        message={errorInfo.message}
      />
    </div>
  );
};

export default TaskSpecModal;
