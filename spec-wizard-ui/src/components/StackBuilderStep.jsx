import React, { useState } from 'react';
import { 
  Library, 
  Plus, 
  Trash2, 
  CheckCircle2, 
  Info, 
  ChevronDown, 
  Code2, 
  Zap, 
  Box,
  Eye,
  EyeOff
} from 'lucide-react';

export default function StackBuilderStep({ formData, setFormData, t, stackTemplates = [] }) {
  const [newLib, setNewLib] = useState({ name: '', mandatory: true, usage_example: '' });
  const [showAddForm, setShowAddForm] = useState(false);

  const applyTemplate = (templateId) => {
    if (templateId === '') {
      setFormData({
        ...formData,
        stack: { id: '', name: '', libraries: [] }
      });
      return;
    }

    const template = stackTemplates.find(t => t.id === templateId);
    if (template) {
      setFormData({
        ...formData,
        stack: {
          id: template.id,
          name: template.name,
          libraries: [...template.libraries]
        }
      });
    }
  };

  const addLibrary = () => {
    if (!newLib.name.trim()) return;
    
    const updatedLibraries = [...(formData.stack?.libraries || []), { ...newLib }];
    setFormData({
      ...formData,
      stack: {
        ...(formData.stack || { id: 'custom', name: 'Custom' }),
        libraries: updatedLibraries
      }
    });
    setNewLib({ name: '', mandatory: true, usage_example: '' });
    setShowAddForm(false);
  };

  const removeLibrary = (index) => {
    const updatedLibraries = formData.stack.libraries.filter((_, i) => i !== index);
    setFormData({
      ...formData,
      stack: {
        ...formData.stack,
        libraries: updatedLibraries
      }
    });
  };

  const toggleMandatory = (index) => {
    const updatedLibraries = [...formData.stack.libraries];
    updatedLibraries[index].mandatory = !updatedLibraries[index].mandatory;
    setFormData({
      ...formData,
      stack: {
        ...formData.stack,
        libraries: updatedLibraries
      }
    });
  };
  const toggleLibraryStatus = (index) => {
    const updatedLibraries = [...formData.stack.libraries];
    updatedLibraries[index].disabled = !updatedLibraries[index].disabled;
    setFormData({
      ...formData,
      stack: {
        ...formData.stack,
        libraries: updatedLibraries
      }
    });
  };

  const currentLibraries = formData.stack?.libraries || [];

  return (
    <div className="space-y-6 animate-in fade-in slide-in-from-right-4 duration-300">
      <section>
        <div className="flex items-center gap-3 mb-4">
          <div className="p-1.5 bg-indigo-50 text-indigo-600 rounded-lg">
            <Box size={18} />
          </div>
          <h3 className="text-md font-bold text-slate-800">{t('stack_configuration')}</h3>
        </div>

        <div className="bg-slate-50/50 p-6 rounded-3xl border border-slate-100 space-y-6">
          {/* Seleção de Template */}
          <div>
            <label className="block text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2">
              {t('stack_plugin_select')}
            </label>
            <div className="relative group">
              <select
                className="w-full px-4 py-3 bg-white border border-slate-200 rounded-2xl outline-none focus:ring-2 focus:ring-indigo-500 transition-all font-bold text-slate-700 appearance-none shadow-sm cursor-pointer"
                value={formData.stack?.id || ''}
                onChange={(e) => applyTemplate(e.target.value)}
              >
                <option value="">{t('stack_plugin_none')}</option>
                {stackTemplates.map(tpl => (
                  <option key={tpl.id} value={tpl.id}>{tpl.name}</option>
                ))}
                {formData.stack?.id === 'custom' && <option value="custom">✨ Custom Stack</option>}
              </select>
              <ChevronDown size={18} className="absolute right-4 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none group-hover:text-indigo-500 transition-colors" />
            </div>
            <p className="mt-2 text-[10px] text-slate-400 flex items-center gap-1">
              <Info size={12} /> {t('stack_desc_info')}
            </p>
          </div>

          <hr className="border-slate-100" />

          {/* Manifesto de Bibliotecas */}
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <h4 className="text-[10px] font-black text-slate-400 uppercase tracking-widest">
                {t('libraries_manifest')}
              </h4>
              <button
                type="button"
                onClick={() => setShowAddForm(!showAddForm)}
                className="flex items-center gap-1.5 px-3 py-1.5 bg-indigo-600 text-white rounded-xl text-[10px] font-bold hover:bg-indigo-700 transition-all shadow-md shadow-indigo-100"
              >
                <Plus size={14} /> {t('add_library')}
              </button>
            </div>

            {/* Lista de Bibliotecas */}
            <div className="grid grid-cols-1 gap-3">
              {currentLibraries.length === 0 ? (
                <div className="py-8 text-center bg-white/50 rounded-2xl border border-dashed border-slate-200">
                  <Library className="w-8 h-8 text-slate-200 mx-auto mb-2" />
                  <p className="text-xs text-slate-400 font-medium">{t('no_libraries')}</p>
                </div>
              ) : (
                currentLibraries.map((lib, idx) => (
                  <div 
                    key={idx} 
                    className={`group bg-white p-4 rounded-2xl border transition-all relative overflow-hidden ${
                      lib.disabled 
                        ? 'opacity-40 grayscale border-slate-100 bg-slate-50/50' 
                        : 'border-slate-200 hover:border-indigo-200 hover:shadow-sm'
                    }`}
                  >
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center gap-2">
                        <div className={`w-2 h-2 rounded-full ${
                          lib.disabled ? 'bg-slate-300' : (lib.mandatory ? 'bg-rose-500 shadow-[0_0_8px_rgba(244,63,94,0.4)]' : 'bg-slate-300')
                        }`} />
                        <span className={`text-sm font-bold ${lib.disabled ? 'text-slate-400 line-through' : 'text-slate-800'}`}>
                          {lib.name}
                        </span>
                        {lib.mandatory && !lib.disabled && (
                          <span className="text-[8px] bg-rose-50 text-rose-600 px-1.5 py-0.5 rounded font-black uppercase tracking-tighter">
                            Mandatory
                          </span>
                        )}
                        {lib.disabled && (
                          <span className="text-[8px] bg-slate-100 text-slate-500 px-1.5 py-0.5 rounded font-black uppercase tracking-tighter">
                            Disabled
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <button
                          onClick={() => toggleLibraryStatus(idx)}
                          className={`p-1.5 rounded-lg transition-all ${lib.disabled ? 'text-slate-600 bg-slate-200' : 'text-slate-400 hover:bg-indigo-50 hover:text-indigo-600'}`}
                          title={lib.disabled ? 'Enable' : 'Disable'}
                        >
                          {lib.disabled ? <EyeOff size={14} /> : <Eye size={14} />}
                        </button>
                        {!lib.disabled && (
                          <button
                            onClick={() => toggleMandatory(idx)}
                            className={`p-1.5 rounded-lg transition-all ${lib.mandatory ? 'text-rose-600 bg-rose-50' : 'text-slate-400 hover:bg-slate-50'}`}
                            title={t('lib_mandatory')}
                          >
                            <CheckCircle2 size={14} />
                          </button>
                        )}
                        <button
                          onClick={() => removeLibrary(idx)}
                          className="p-1.5 text-slate-400 hover:text-rose-600 hover:bg-rose-50 rounded-lg transition-all"
                        >
                          <Trash2 size={14} />
                        </button>
                      </div>
                    </div>
                    {lib.usage_example && (
                      <div className={`p-3 rounded-xl border mt-2 ${lib.disabled ? 'bg-slate-200/20 border-slate-100' : 'bg-slate-900/5 border-slate-100'}`}>
                        <div className="flex items-center gap-1.5 text-[9px] font-black text-slate-400 uppercase mb-2">
                          <Code2 size={10} /> Usage Example
                        </div>
                        <pre className={`text-[10px] font-mono overflow-x-auto whitespace-pre-wrap leading-relaxed p-2 rounded-lg border ${
                          lib.disabled ? 'text-slate-400 border-transparent bg-transparent' : 'text-slate-600 border-white/80 bg-white/50'
                        }`}>
                          {lib.usage_example}
                        </pre>
                      </div>
                    )}
                  </div>
                ))
              )}
            </div>

            {/* Formulário de Nova Biblioteca */}
            {showAddForm && (
              <div className="p-5 bg-white rounded-3xl border-2 border-indigo-100 shadow-xl animate-in zoom-in-95 duration-200">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                  <div>
                    <label className="block text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2">
                      {t('lib_name')}
                    </label>
                    <input
                      type="text"
                      className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl outline-none focus:ring-2 focus:ring-indigo-500 font-medium text-sm"
                      value={newLib.name}
                      onChange={(e) => setNewLib({ ...newLib, name: e.target.value })}
                      placeholder="Ex: dio, axios, zap"
                    />
                  </div>
                  <div className="flex items-end pb-1">
                    <label className="flex items-center gap-2 cursor-pointer group">
                      <input
                        type="checkbox"
                        className="hidden"
                        checked={newLib.mandatory}
                        onChange={(e) => setNewLib({ ...newLib, mandatory: e.target.checked })}
                      />
                      <div className={`w-5 h-5 rounded-md border-2 flex items-center justify-center transition-all ${newLib.mandatory ? 'bg-indigo-600 border-indigo-600 shadow-lg shadow-indigo-100' : 'border-slate-300'}`}>
                        {newLib.mandatory && <CheckCircle2 size={14} className="text-white" />}
                      </div>
                      <span className="text-xs font-bold text-slate-600 group-hover:text-indigo-600 transition-colors">
                        {t('lib_mandatory')}
                      </span>
                    </label>
                  </div>
                </div>
                <div>
                  <label className="block text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2">
                    {t('lib_example')}
                  </label>
                  <textarea
                    className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl outline-none focus:ring-2 focus:ring-indigo-500 font-mono text-[10px] min-h-[100px]"
                    value={newLib.usage_example}
                    onChange={(e) => setNewLib({ ...newLib, usage_example: e.target.value })}
                    placeholder={t('lib_usage_placeholder')}
                  />
                </div>
                <div className="flex justify-end gap-3 mt-4">
                  <button
                    onClick={() => setShowAddForm(false)}
                    className="px-4 py-2 text-xs font-bold text-slate-500 hover:text-slate-700 transition-all"
                  >
                    {t('cancel')}
                  </button>
                  <button
                    onClick={addLibrary}
                    className="px-6 py-2 bg-slate-900 text-white rounded-xl text-xs font-bold hover:bg-indigo-600 transition-all shadow-lg"
                  >
                    {t('add')}
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      </section>
    </div>
  );
}
