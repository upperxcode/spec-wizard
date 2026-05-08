import React, { useState, useEffect } from 'react';
import { 
  Link as LinkIcon, 
  FileUp, 
  FileText, 
  Globe, 
  Trash2, 
  Loader2, 
  CheckCircle2, 
  AlertCircle,
  Clock,
  ExternalLink,
  Plus,
  BookOpen
} from 'lucide-react';

export function KnowledgeBase({ t, apiRequest, projectPath }) {
  const [sources, setSources] = useState([]);
  const [loading, setLoading] = useState(false);
  const [newLink, setNewLink] = useState('');
  const [isUploading, setIsUploading] = useState(false);

  const fetchKnowledge = async () => {
    if (!projectPath) return;
    setLoading(true);
    try {
      const response = await apiRequest(`/knowledge?path=${encodeURIComponent(projectPath)}`);
      const data = await response.json();
      setSources(data || []);
    } catch (err) {
      console.error("Erro ao buscar base de conhecimento:", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchKnowledge();
    // Poll for status updates
    const interval = setInterval(fetchKnowledge, 10000);
    return () => clearInterval(interval);
  }, [projectPath]);

  const handleAddLink = async (e) => {
    e.preventDefault();
    if (!newLink) return;
    
    setLoading(true);
    try {
      const response = await apiRequest('/knowledge/link', {
        method: 'POST',
        body: JSON.stringify({
          path: projectPath,
          url: newLink
        })
      });
      if (response.ok) {
        setNewLink('');
        fetchKnowledge();
      }
    } catch (err) {
      console.error("Erro ao adicionar link:", err);
    } finally {
      setLoading(false);
    }
  };

  const handleFileUpload = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    setIsUploading(true);
    const formData = new FormData();
    formData.append('file', file);
    formData.append('path', projectPath);

    try {
      const response = await apiRequest('/knowledge/upload', {
        method: 'POST',
        body: formData
      });
      if (response.ok) {
        fetchKnowledge();
      }
    } catch (err) {
      console.error("Erro no upload:", err);
    } finally {
      setIsUploading(false);
    }
  };

  const handleDelete = async (id) => {
    if (!window.confirm(t('confirm_delete_source'))) return;
    
    setLoading(true);
    try {
      const response = await apiRequest(`/knowledge?path=${encodeURIComponent(projectPath)}&id=${id}`, {
        method: 'DELETE'
      });
      if (response.ok) {
        fetchKnowledge();
      }
    } catch (err) {
      console.error("Erro ao deletar fonte:", err);
    } finally {
      setLoading(false);
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'processed':
        return <CheckCircle2 className="w-4 h-4 text-emerald-500" />;
      case 'processing':
        return <Loader2 className="w-4 h-4 text-blue-500 animate-spin" />;
      case 'error':
        return <AlertCircle className="w-4 h-4 text-rose-500" />;
      default:
        return <Clock className="w-4 h-4 text-slate-400" />;
    }
  };

  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-3">
          <div className="w-12 h-12 bg-blue-50 rounded-2xl flex items-center justify-center">
            <BookOpen className="w-6 h-6 text-blue-600" />
          </div>
          <div>
            <h2 className="text-2xl font-bold text-slate-900">
              {t('knowledge_base')}
            </h2>
            <p className="text-slate-500 text-sm">
              {t('knowledge_base_desc')}
            </p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Add Link */}
        <form onSubmit={handleAddLink} className="bg-white p-6 rounded-3xl border border-slate-200/60 shadow-sm flex flex-col gap-4">
          <label className="text-sm font-bold text-slate-700 flex items-center gap-2">
            <LinkIcon className="w-4 h-4 text-blue-500" />
            {t('add_link')}
          </label>
          <div className="flex gap-2">
            <input
              type="url"
              value={newLink}
              onChange={(e) => setNewLink(e.target.value)}
              placeholder={t('url_placeholder')}
              className="flex-1 bg-slate-50 border border-slate-200 rounded-2xl px-4 py-3 text-sm focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-700"
            />
            <button
              type="submit"
              disabled={loading || !newLink}
              className="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 px-4 rounded-2xl transition-all shadow-lg shadow-blue-200"
            >
              <Plus className="w-5 h-5 text-white" />
            </button>
          </div>
        </form>

        {/* Upload File */}
        <div className="bg-white p-6 rounded-3xl border border-slate-200/60 shadow-sm flex flex-col gap-4">
          <label className="text-sm font-bold text-slate-700 flex items-center gap-2">
            <FileUp className="w-4 h-4 text-emerald-500" />
            {t('upload_file')}
          </label>
          <div className="relative group">
            <input
              type="file"
              onChange={handleFileUpload}
              className="absolute inset-0 w-full h-full opacity-0 cursor-pointer z-10"
              accept=".pdf,.xlsx,.xls,.csv"
            />
            <div className={`w-full border-2 border-dashed ${isUploading ? 'border-blue-500 bg-blue-50' : 'border-slate-200 group-hover:border-blue-400 group-hover:bg-slate-50'} rounded-2xl p-4 flex items-center justify-center gap-3 transition-all`}>
              {isUploading ? (
                <>
                  <Loader2 className="w-5 h-5 text-blue-600 animate-spin" />
                  <span className="text-sm text-blue-600 font-bold">Enviando...</span>
                </>
              ) : (
                <>
                  <FileUp className="w-5 h-5 text-slate-400 group-hover:text-blue-500" />
                  <span className="text-sm text-slate-500 font-medium">PDF, Excel, CSV</span>
                </>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Sources Table */}
      <div className="bg-white rounded-3xl border border-slate-200/60 shadow-sm overflow-hidden">
        <div className="p-6 border-b border-slate-100 flex items-center justify-between">
          <h3 className="font-bold text-slate-900">{t('knowledge_sources')}</h3>
          <span className="text-[10px] font-black uppercase tracking-widest bg-slate-100 px-3 py-1 rounded-full text-slate-500">
            {sources.length} Total
          </span>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead>
              <tr className="text-[10px] font-black text-slate-400 uppercase tracking-widest border-b border-slate-50">
                <th className="px-6 py-4">{t('source_path')}</th>
                <th className="px-6 py-4">{t('source_type')}</th>
                <th className="px-6 py-4">{t('source_status')}</th>
                <th className="px-6 py-4 text-right">Ação</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50">
              {sources.length === 0 ? (
                <tr>
                  <td colSpan="4" className="px-6 py-12 text-center text-slate-400 italic">
                    {t('no_sources')}
                  </td>
                </tr>
              ) : (
                sources.map((source, index) => (
                  <tr key={index} className="group hover:bg-slate-50/50 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3 max-w-md">
                        <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${source.type === 'link' ? 'bg-blue-50' : 'bg-emerald-50'}`}>
                          {source.type === 'link' ? <Globe className="w-4 h-4 text-blue-600 shrink-0" /> : <FileText className="w-4 h-4 text-emerald-600 shrink-0" />}
                        </div>
                        <span className="text-sm text-slate-700 truncate font-semibold">
                          {source.path}
                        </span>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <span className="text-[10px] font-black px-2 py-1 rounded-md bg-slate-100 text-slate-500 uppercase tracking-tighter">
                        {source.type}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        {getStatusIcon(source.status)}
                        <span className={`text-[10px] font-black uppercase tracking-tighter ${
                          source.status === 'processed' ? 'text-emerald-600' : 
                          source.status === 'error' ? 'text-rose-600' : 
                          source.status === 'processing' ? 'text-blue-600' : 'text-slate-400'
                        }`}>
                          {t(`status_${source.status}`)}
                        </span>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-right">
                      <div className="flex items-center justify-end gap-2 opacity-0 group-hover:opacity-100 transition-all">
                        {source.type === 'link' && (
                          <a 
                            href={source.path} 
                            target="_blank" 
                            rel="noopener noreferrer"
                            className="p-2 hover:bg-blue-50 rounded-xl text-slate-400 hover:text-blue-600 transition-colors"
                          >
                            <ExternalLink className="w-4 h-4" />
                          </a>
                        )}
                        <button 
                          onClick={() => handleDelete(source.id)}
                          className="p-2 hover:bg-rose-50 rounded-xl text-slate-400 hover:text-rose-600 transition-colors"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
