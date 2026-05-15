import { motion } from 'framer-motion';
import { AlertTriangle, ShieldAlert, ShieldCheck } from 'lucide-react';

export function AnomalyList({ anomalies }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: 0.4 }}
      className="bg-white/80 backdrop-blur-xl rounded-2xl p-6 shadow-sm border border-white/50 mt-6"
    >
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-sm font-semibold text-gray-800 tracking-wide uppercase">Detected Anomalies</h3>
        {anomalies?.length > 0 ? (
          <span className="px-3 py-1 bg-red-100 text-red-600 text-xs font-bold rounded-full">
            {anomalies.length} Issues
          </span>
        ) : (
          <span className="px-3 py-1 bg-emerald-100 text-emerald-600 text-xs font-bold rounded-full flex items-center gap-1">
            <ShieldCheck className="w-3 h-3" /> Clean
          </span>
        )}
      </div>

      {(!anomalies || anomalies.length === 0) ? (
        <div className="flex flex-col items-center justify-center py-10 bg-gray-50/50 rounded-xl border border-dashed border-gray-200">
          <ShieldCheck className="w-12 h-12 text-emerald-400 mb-3" />
          <p className="text-sm font-medium text-gray-900">No anomalies detected</p>
          <p className="text-xs text-gray-500 mt-1">Your dataset looks perfectly clean</p>
        </div>
      ) : (
        <div className="space-y-3 max-h-[400px] overflow-y-auto pr-2 custom-scrollbar">
          {anomalies.map((a, i) => (
            <motion.div
              key={i}
              initial={{ opacity: 0, scale: 0.95, x: -10 }}
              animate={{ opacity: 1, scale: 1, x: 0 }}
              transition={{ delay: 0.5 + (i * 0.05), type: "spring" }}
              className={`flex items-start gap-4 p-4 rounded-xl border relative overflow-hidden ${
                a.severity === 'high' 
                  ? 'bg-red-50/50 border-red-100 hover:bg-red-50' 
                  : 'bg-amber-50/50 border-amber-100 hover:bg-amber-50'
              } transition-colors`}
            >
              <div className={`absolute left-0 top-0 bottom-0 w-1 ${
                a.severity === 'high' ? 'bg-red-500' : 'bg-amber-500'
              }`} />
              
              <div className={`p-2 rounded-lg ${
                a.severity === 'high' ? 'bg-red-100/50' : 'bg-amber-100/50'
              }`}>
                {a.severity === 'high' ? (
                  <ShieldAlert className={`w-5 h-5 text-red-500`} />
                ) : (
                  <AlertTriangle className={`w-5 h-5 text-amber-500`} />
                )}
              </div>
              
              <div className="flex-1 min-w-0 pt-0.5">
                <p className="text-sm font-bold text-gray-900 mb-0.5">{a.anomaly_type}</p>
                <p className="text-sm text-gray-600 leading-relaxed">{a.description}</p>
              </div>
            </motion.div>
          ))}
        </div>
      )}
    </motion.div>
  );
}
