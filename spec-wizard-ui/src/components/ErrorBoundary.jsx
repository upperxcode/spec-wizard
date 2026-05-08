import React from 'react';
import { AlertTriangle, RefreshCcw, Home } from 'lucide-react';

class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true };
  }

  componentDidCatch(error, errorInfo) {
    this.setState({ error, errorInfo });
    console.error("ErrorBoundary caught an error:", error, errorInfo);
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null, errorInfo: null });
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      return (
        <div className="min-h-screen bg-slate-50 flex items-center justify-center p-6 font-sans">
          <div className="max-w-md w-full bg-white rounded-[40px] border border-slate-200/60 shadow-xl p-10 text-center animate-in zoom-in-95 duration-300">
            <div className="w-20 h-20 bg-amber-50 rounded-3xl flex items-center justify-center mx-auto mb-8 border border-amber-100/50">
              <AlertTriangle size={40} className="text-amber-500" />
            </div>
            
            <h1 className="text-3xl font-bold text-slate-900 mb-4 tracking-tight">
              Oops! Algo deu errado
            </h1>
            
            <p className="text-slate-600 mb-8 leading-relaxed">
              Ocorreu um erro inesperado na interface. Mas não se preocupe, seus dados no servidor estão seguros.
            </p>

            <div className="bg-slate-50 rounded-2xl p-4 mb-8 text-left border border-slate-100 overflow-hidden">
              <p className="text-[10px] font-mono text-slate-400 uppercase tracking-widest mb-2 font-bold">Detalhes Técnicos</p>
              <div className="text-xs font-mono text-slate-500 break-words line-clamp-3">
                {this.state.error?.toString() || "Erro desconhecido"}
              </div>
            </div>

            <div className="flex flex-col gap-3">
              <button
                onClick={this.handleReset}
                className="w-full bg-[#aa3bff] hover:bg-[#9032e6] text-white font-bold py-4 rounded-2xl transition-all flex items-center justify-center gap-2 shadow-lg shadow-purple-200 active:scale-95"
              >
                <RefreshCcw size={18} /> Tentar Novamente
              </button>
              
              <button
                onClick={() => window.location.href = '/'}
                className="w-full bg-slate-100 hover:bg-slate-200 text-slate-600 font-bold py-4 rounded-2xl transition-all flex items-center justify-center gap-2 active:scale-95"
              >
                <Home size={18} /> Voltar ao Início
              </button>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
