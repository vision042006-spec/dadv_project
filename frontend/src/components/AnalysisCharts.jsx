import { motion } from 'framer-motion';

const CHART_COLORS = ['#10b981', '#f59e0b', '#ef4444', '#6366f1', '#8b5cf6'];

export function AnalysisCharts({ data }) {
  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ delay: 0.2 }}
        className="bg-white/80 backdrop-blur-xl rounded-2xl p-6 shadow-sm border border-white/50"
      >
        <h3 className="text-sm font-semibold text-gray-800 mb-6 tracking-wide uppercase">File Types</h3>
        <PieChartComponent data={data.fileTypes} />
      </motion.div>
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ delay: 0.3 }}
        className="bg-white/80 backdrop-blur-xl rounded-2xl p-6 shadow-sm border border-white/50"
      >
        <h3 className="text-sm font-semibold text-gray-800 mb-6 tracking-wide uppercase">Size Distribution</h3>
        <BarChartComponent data={data.sizeDist} />
      </motion.div>
    </div>
  );
}

function PieChartComponent({ data }) {
  if (!data || data.length === 0) {
    return (
      <div className="h-48 flex items-center justify-center">
        <p className="text-sm text-gray-400">No data available</p>
      </div>
    );
  }

  const maxPercent = Math.max(...data.map(d => d.percent || 0));

  return (
    <div className="space-y-4">
      {data.slice(0, 5).map((d, i) => (
        <div key={i} className="group">
          <div className="flex items-center justify-between mb-1.5">
            <div className="flex items-center gap-2">
              <div 
                className="w-2.5 h-2.5 rounded-full" 
                style={{ backgroundColor: CHART_COLORS[i % CHART_COLORS.length] }} 
              />
              <span className="text-sm font-medium text-gray-700">{d.file_type || 'unknown'}</span>
            </div>
            <div className="flex items-center gap-3">
              <span className="text-xs text-gray-400">{d.count} files</span>
              <span className="text-sm font-semibold text-gray-900 w-12 text-right">
                {typeof d.percent === 'number' ? d.percent.toFixed(1) : '0.0'}%
              </span>
            </div>
          </div>
          <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
            <motion.div
              className="h-full rounded-full transition-all group-hover:opacity-80"
              style={{ backgroundColor: CHART_COLORS[i % CHART_COLORS.length] }}
              initial={{ width: 0 }}
              animate={{ width: `${(d.percent / maxPercent) * 100}%` }}
              transition={{ duration: 1, delay: 0.4 + (i * 0.1), type: "spring" }}
            />
          </div>
        </div>
      ))}
    </div>
  );
}

function BarChartComponent({ data }) {
  if (!data || data.length === 0) {
    return (
      <div className="h-48 flex items-center justify-center">
        <p className="text-sm text-gray-400">No data available</p>
      </div>
    );
  }

  const max = Math.max(...data.map(d => d.count));

  return (
    <div className="space-y-4 h-full flex flex-col justify-center">
      {data.map((d, i) => (
        <div key={i} className="group">
          <div className="flex justify-between text-sm mb-1.5">
            <span className="font-medium text-gray-700">{d.bucket}</span>
            <span className="font-semibold text-gray-900">{d.count}</span>
          </div>
          <div className="h-3 bg-gray-100 rounded-full overflow-hidden">
            <motion.div
              className="h-full bg-gradient-to-r from-emerald-400 to-emerald-500 rounded-full shadow-inner"
              initial={{ width: 0 }}
              animate={{ width: `${(d.count / max) * 100}%` }}
              transition={{ duration: 0.8, delay: 0.4 + (i * 0.1), ease: "easeOut" }}
            />
          </div>
        </div>
      ))}
    </div>
  );
}
